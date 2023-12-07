package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	src       string
	dist      string
	intervals int
	rootCmd   = &cobra.Command{
		Use:   "cron-backup",
		Short: "定时备份",
		Long:  `将目标目录下的文件定时压缩，并备份到指定目录下`,
		Run: func(cmd *cobra.Command, args []string) {
			// 判断dist目录是否存在，如果不存在则创建
			_, err := os.Stat(dist)
			if !os.IsExist(err) {
				os.MkdirAll(dist, os.ModePerm)
			}
			// 服务启动时先执行一次
			dst := filepath.Join(dist, time.Now().Format("2006-01-02 15:04:05")+".zip")
			if err := Zip(dst, src); err != nil {
				log.Fatalln(err)
			}
			// 启动一个定时器，定时将给定目录下的文件压缩到指定目录下
			timeTicker := time.Tick(time.Duration(intervals) * time.Second)
			for {
				select {
				case <-timeTicker:
					dst := filepath.Join(dist, time.Now().Format("2006-01-02 15:04:05")+".zip")
					if err := Zip(dst, src); err != nil {
						log.Fatalln(err)
					}
				}
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&src, "s", "", "源目录")
	rootCmd.PersistentFlags().StringVar(&dist, "d", "", "目标目录")
	rootCmd.PersistentFlags().IntVar(&intervals, "i", 3600, "备份时间间隔")
}

func main() {
	rootCmd.Execute()
}

func Zip(dst, src string) (err error) {
	// 创建准备写入的文件
	fw, err := os.Create(dst)
	defer fw.Close()
	if err != nil {
		return err
	}

	// 通过 fw 来创建 zip.Write
	zw := zip.NewWriter(fw)
	defer func() {
		// 检测一下是否成功关闭
		if err := zw.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// 下面来将文件写入 zw ，因为有可能会有很多个目录及文件，所以递归处理
	return filepath.Walk(src, func(path string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}

		// 通过文件信息，创建 zip 的文件信息
		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return
		}

		// 写入文件信息，并返回一个 Write 结构
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return
		}

		// 检测，如果不是标准文件就只写入头信息，不写入文件数据到 w
		// 如目录，也没有数据需要写
		if !fh.Mode().IsRegular() {
			return nil
		}

		// 打开要压缩的文件
		fr, err := os.Open(path)
		defer fr.Close()
		if err != nil {
			return
		}

		// 将打开的文件 Copy 到 w
		n, err := io.Copy(w, fr)
		if err != nil {
			return
		}
		// 输出压缩的内容
		fmt.Printf("成功压缩文件： %s, 共写入了 %d 个字符的数据\n", path, n)

		return nil
	})
}
