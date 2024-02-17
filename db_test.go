package dbcmp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func Test_lengthOfLIS(t *testing.T) {
	t.Run("case1", func(t *testing.T) {
		input := []int{10, 9, 2, 5, 3, 7, 101, 18}
		expect := 4
		got := lengthOfLIS(input)
		require.Equal(t, expect, got)
	})
	t.Run("case2", func(t *testing.T) {
		input := []int{0, 1, 0, 3, 2, 3}
		expect := 4
		got := lengthOfLIS(input)
		require.Equal(t, expect, got)
	})

	t.Run("case3", func(t *testing.T) {
		input := []int{4, 10, 4, 3, 8, 9}
		expect := 3
		got := lengthOfLIS(input)
		require.Equal(t, expect, got)
	})
}

func lengthOfLIS(nums []int) int {
	sub := []int{nums[0]}

	for i := 1; i < len(nums); i++ {
		num := nums[i]
		if num > sub[len(sub)-1] {
			sub = append(sub, num)
		} else {
			j := sort.Search(len(sub), func(m int) bool { return sub[m] >= num })
			sub[j] = num
		}
	}
	return len(sub)
}

func BenchmarkBbolt(b *testing.B) {
	bktName := []byte("bktName")

	fillDb := func(db *bbolt.DB, data [][]byte) error {
		buf := make([]byte, 8)

		return db.Update(func(tx *bbolt.Tx) error {
			bkt, err := tx.CreateBucketIfNotExists(bktName)
			if err != nil {
				return err
			}

			for i, chunk := range data {
				binary.BigEndian.PutUint64(buf, uint64(i))
				err = bkt.Put(buf, chunk)
				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	clearDb := func(db *bbolt.DB) error {
		return db.Update(func(tx *bbolt.Tx) error {
			return tx.DeleteBucket(bktName)
		})
	}

	readDb := func(db *bbolt.DB, data [][]byte) error {
		buf := make([]byte, 8)

		return db.Update(func(tx *bbolt.Tx) error {
			bkt, err := tx.CreateBucketIfNotExists(bktName)
			if err != nil {
				return err
			}

			for i, expect := range data {
				binary.BigEndian.PutUint64(buf, uint64(i))
				val := bkt.Get(buf)
				if !bytes.Equal(expect, val) {
					b.Fatal("Values not equal for key " + string(val))
				}
			}

			return nil
		})
	}

	runWriteBench := func(n, ch int) func(*testing.B) {
		return func(b *testing.B) {
			db := testBbolt(b)

			data := generateBytes(n, ch)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := fillDb(db, data); err != nil {
					b.Fatal(err.Error())
				}
				if err := clearDb(db); err != nil {
					b.Fatal(err.Error())
				}
			}
		}
	}

	runReadBench := func(n, ch int) func(*testing.B) {
		return func(b *testing.B) {
			db := testBbolt(b)

			data := generateBytes(n, ch)

			if err := fillDb(db, data); err != nil {
				b.Fatal(err.Error())
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := readDb(db, data); err != nil {
					b.Fatal(err.Error())
				}
			}
		}
	}

	b.Run("write", func(b *testing.B) {
		b.Run("100 chunks by 500 symbols", runWriteBench(100, 500))
		b.Run("1000 chunks by 500 symbols", runWriteBench(1000, 500))
		b.Run("10_000 chunks by 500 symbols", runWriteBench(10_000, 500))
		// b.Run("30_000 chunks by 500 symbols", runWriteBench(30_000, 500))
		// b.Run("50_000 chunks by 500 symbols", runWriteBench(50_000, 500))
		// b.Run("100_000 chunks by 500 symbols", runWriteBench(100_000, 500))

		b.Run("100 chunks by 10_000 symbols", runWriteBench(100, 10_000))
		b.Run("1000 chunks by 10_000 symbols", runWriteBench(1000, 10_000))
		b.Run("10_000 chunks by 10_000 symbols", runWriteBench(10_000, 10_000))
		// b.Run("30_000 chunks by 10_000 symbols", runWriteBench(30_000, 10_000))
		// b.Run("50_000 chunks by 10_000 symbols", runWriteBench(50_000, 10_000))
		// b.Run("100_000 chunks by 10_000 symbols", runWriteBench(100_000, 10_000))
	})

	b.Run("read", func(b *testing.B) {
		b.Run("100 chunks by 500 symbols", runReadBench(100, 500))
		// b.Run("1000 chunks by 500 symbols", runReadBench(1000, 500))
		// b.Run("10_000 chunks by 500 symbols", runReadBench(10_000, 500))
		// b.Run("30_000 chunks by 500 symbols", runReadBench(30_000, 500))
		// b.Run("50_000 chunks by 500 symbols", runReadBench(50_000, 500))
		// b.Run("100_000 chunks by 500 symbols", runReadBench(100_000, 500))
	})
}

func BenchmarkBadger(b *testing.B) {

	fillDb := func(db *badger.DB, data [][]byte) error {
		const batchSize = 10_000
		for i := 0; i < len(data); i += batchSize {
			from := i
			to := i + batchSize
			if to > len(data) {
				to = len(data)
			}
			batch := data[from:to]
			err := db.Update(func(txn *badger.Txn) error {
				for j, chunk := range batch {
					key := make([]byte, 8)
					binary.BigEndian.PutUint64(key, uint64(i+j))

					err := txn.Set(key, chunk)
					if err != nil {
						return err
					}
				}

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	}

	clearDb := func(db *badger.DB, data [][]byte) error {
		return db.Update(func(txn *badger.Txn) error {
			for i, chunk := range data {
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, uint64(i))

				err := txn.Set(key, chunk)
				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	readDb := func(db *badger.DB, data [][]byte) error {
		return db.View(func(txn *badger.Txn) error {
			for i, chunk := range data {
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, uint64(i))

				item, err := txn.Get(key)
				if err != nil {
					return err
				}
				err = item.Value(func(val []byte) error {
					if !bytes.Equal(val, chunk) {
						return errors.New("not equal")
					}

					return nil
				})

				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	runWriteBench := func(n, ch int) func(*testing.B) {
		return func(b *testing.B) {
			db := testBadger(b)

			data := generateBytes(n, ch)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := fillDb(db, data); err != nil {
					b.Fatal(err.Error())
				}
				if err := clearDb(db, data); err != nil {
					b.Fatal(err.Error())
				}
			}
		}
	}

	runReadBench := func(n, ch int) func(*testing.B) {
		return func(b *testing.B) {
			db := testBadger(b)

			data := generateBytes(n, ch)

			if err := fillDb(db, data); err != nil {
				b.Fatal(err.Error())
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := readDb(db, data); err != nil {
					b.Fatal(err.Error())
				}
			}
		}
	}

	b.Run("write", func(b *testing.B) {
		b.Run("100 chunks by 500 symbols", runWriteBench(100, 500))
		b.Run("1000 chunks by 500 symbols", runWriteBench(1000, 500))
		b.Run("10_000 chunks by 500 symbols", runWriteBench(10_000, 500))
		b.Run("30_000 chunks by 500 symbols", runWriteBench(30_000, 500))
		b.Run("50_000 chunks by 500 symbols", runWriteBench(50_000, 500))
		b.Run("100_000 chunks by 500 symbols", runWriteBench(100_000, 500))
	})

	b.Run("read", func(b *testing.B) {
		b.Run("100 chunks by 500 symbols", runReadBench(100, 500))
		// b.Run("1000 chunks by 500 symbols", runReadBench(1000, 500))
		// b.Run("10_000 chunks by 500 symbols", runReadBench(10_000, 500))
		// b.Run("30_000 chunks by 500 symbols", runReadBench(30_000, 500))
		// b.Run("50_000 chunks by 500 symbols", runReadBench(50_000, 500))
		// b.Run("100_000 chunks by 500 symbols", runReadBench(100_000, 500))
	})
}

type noopBadgerLogger struct{}

func (*noopBadgerLogger) Errorf(string, ...interface{})   {}
func (*noopBadgerLogger) Warningf(string, ...interface{}) {}
func (*noopBadgerLogger) Infof(string, ...interface{})    {}
func (*noopBadgerLogger) Debugf(string, ...interface{})   {}

func testBadger(b *testing.B) *badger.DB {
	dbFile := fmt.Sprintf("/tmp/bench-badger-db-%d.db", rand.Int31())

	opts := badger.DefaultOptions(dbFile)
	opts.Logger = &noopBadgerLogger{}
	db, err := badger.Open(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		err := db.Close()
		if err != nil {
			b.Log(err)
		}
		err = os.RemoveAll(dbFile)
		if err != nil {
			b.Log(err.Error())
		}
	})

	return db
}

func testBbolt(b *testing.B) *bbolt.DB {
	dbFile := fmt.Sprintf("/tmp/bench-bbolt-db-%d.db", rand.Int31())

	db, err := bbolt.Open(dbFile, os.ModePerm, bbolt.DefaultOptions)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		err := db.Close()
		if err != nil {
			b.Log(err)
		}
		err = os.Remove(dbFile)
		if err != nil {
			b.Log(err.Error())
		}
	})

	return db

}

func generateBytes(n, chunkSize int) [][]byte {
	res := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		chunk := make([]byte, 0, chunkSize)
		for j := 0; j < chunkSize; j++ {
			chunk = append(chunk, randomSymbol())
		}

		res = append(res, chunk)
	}

	return res
}

func randomSymbol() byte {
	n := rand.Intn(128)
	return byte(n)
}
