package isureal

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/jmoiron/sqlx"
)

// PrewarmImageDiskCache は初期データの画像を、アプリと同じ場所・同じ形式でディスクに書いておく（#65）。
// 初期データは約 183万枚あり、ベンチは毎回ほぼ別の画像を読むので、読んだものを覚えるだけでは当たらない。
// ベンチの外で一度だけ流す（s4 の読み込みが重い）。既にあるファイルは書き直さない。
func PrewarmImageDiskCache(ctx context.Context, config *Config, parallel int) error {
	db, err := sqlx.Open("pgx", config.DatabaseDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	var parts []string
	if err := db.SelectContext(ctx, &parts, `
  SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
  WHERE i.inhparent = 'image_objects'::regclass AND c.relkind = 'r'
  ORDER BY c.relname`); err != nil {
		return err
	}
	var written, total atomic.Int64
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	errs := make(chan error, len(parts))
	for _, part := range parts {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			rows, err := db.QueryxContext(ctx, fmt.Sprintf(`SELECT id, data FROM %s WHERE created_at <= (SELECT created_at FROM initial_data_boundary)`, part), binaryImage)
			if err != nil {
				errs <- err
				return
			}
			defer rows.Close()
			n := 0
			for rows.Next() {
				var id string
				var data []byte
				if err := rows.Scan(&id, &data); err != nil {
					errs <- err
					return
				}
				total.Add(1)
				n++
				if path, ok := imageDiskPath(id); ok {
					if _, err := os.Stat(path); err != nil {
						writeImageFile(id, data)
						written.Add(1)
					}
				}
			}
			if err := rows.Err(); err != nil {
				errs <- err
				return
			}
			fmt.Fprintf(os.Stderr, "%s: %d images (total %d, written %d)\n", part, n, total.Load(), written.Load())
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return err
	}
	fmt.Fprintf(os.Stderr, "done: %d images, %d written\n", total.Load(), written.Load())
	return nil
}
