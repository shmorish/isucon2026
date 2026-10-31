package isureal

import (
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// maxUploadBytes は nginx の client_max_body_size。これより大きい本文はここまで来ない
const maxUploadBytes = 32 << 20

// uploadSlackBytes は Content-Length で確保した画像バッファに残ってよい余り（multipart の境界・ヘッダーと
// 他のフィールド。caption は最大 1000 文字）。これを超える余りはキャッシュに入れる前にコピーする
const uploadSlackBytes = 8 << 10

// readBodyExact は本文を Content-Length の長さで1回だけ確保して読む。io.ReadAll は 512B から倍々に伸ばすので
// 画像1枚ごとに確保・ゼロ埋め・コピーを繰り返していた。長さが分からない・上限を超えるときは io.ReadAll のまま。
// 本文が Content-Length より短いときの io.ErrUnexpectedEOF は io.ReadAll と同じ
func readBodyExact(r *http.Request) ([]byte, error) {
	n := r.ContentLength
	if n <= 0 || n > maxUploadBytes {
		return io.ReadAll(r.Body)
	}
	buf := make([]byte, n)
	read, err := io.ReadFull(r.Body, buf)
	return buf[:read], err
}

// readMultipartUpload は multipart/form-data をパートごとに読み、fileField の最初のファイルパートを
// Content-Length 分だけ確保した1つのバッファに入れて返す（ReadForm は bytes.Buffer を倍々に伸ばし、その後
// FileHeader.Open + io.ReadAll でもう一度コピーしていた）。読んだフォームは Request.MultipartForm に置くので、
// 続く c.Bind（ParseMultipartForm は MultipartForm が入っていれば何もしない）が今まで通りフィールドを束縛する。
// 返す FileHeader に中身はないので、ファイルの中身は戻り値から取ること。
// echo の BindBody が multipart として扱わないリクエスト（Content-Length 0 や別の Content-Type）は nil, nil を返し、
// c.Bind がこれまで通り 415 などを決める。エラーは ParseMultipartForm が返すものと同じ
func readMultipartUpload(r *http.Request, fileField string) ([]byte, error) {
	base, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	if r.ContentLength == 0 || strings.TrimSpace(base) != "multipart/form-data" {
		return nil, nil
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}
	form := &multipart.Form{Value: map[string][]string{}, File: map[string][]*multipart.FileHeader{}}
	var file []byte
	remaining := int64(maxUploadBytes)
	for parts := 0; ; parts++ {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if parts >= 1000 {
			return nil, multipart.ErrMessageTooLarge
		}
		name := p.FormName()
		if name == "" {
			continue
		}
		if p.FileName() == "" {
			value, err := io.ReadAll(io.LimitReader(p, remaining+1))
			if err != nil {
				return nil, err
			}
			remaining -= int64(len(value))
			if remaining < 0 {
				return nil, multipart.ErrMessageTooLarge
			}
			form.Value[name] = append(form.Value[name], string(value))
			continue
		}
		fh := &multipart.FileHeader{Filename: p.FileName(), Header: p.Header}
		if name == fileField && file == nil {
			if file, err = readPartExact(p, r.ContentLength); err != nil {
				return nil, err
			}
			fh.Size = int64(len(file))
		} else {
			if fh.Size, err = io.Copy(io.Discard, p); err != nil {
				return nil, err
			}
		}
		form.File[name] = append(form.File[name], fh)
	}
	// ParseMultipartForm と同じく URL のクエリを Form に入れ、multipart の値を Form と PostForm にも足す
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	for k, v := range form.Value {
		r.Form[k] = append(r.Form[k], v...)
		r.PostForm[k] = append(r.PostForm[k], v...)
	}
	r.MultipartForm = form
	if file == nil {
		file = []byte{}
	}
	return file, nil
}

// readPartExact はパートを本文全体の長さ（パートはそれより必ず短い）で確保したバッファに読む。
// 長さが分からない・上限を超えるときだけ append で伸ばす
func readPartExact(p *multipart.Part, contentLength int64) ([]byte, error) {
	size := contentLength
	if size <= 0 || size > maxUploadBytes {
		size = 512
	}
	buf := make([]byte, 0, size)
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := p.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err == io.EOF {
			return buf, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
