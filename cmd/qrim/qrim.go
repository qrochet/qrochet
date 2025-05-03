// qrim is a small image manipulation command that uses only the Go standard library.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

type qrim struct {
	in      string
	out     string
	target  string
	mask    string
	form    string
	width   int
	height  int
	quality int
}

func (q qrim) resizeImage(output io.Writer, input io.Reader, form string) error {
	src, _, err := image.Decode(input)

	if err != nil {
		return err
	}

	dst := image.NewRGBA(image.Rect(0, 0, q.width, q.height))
	draw.Draw(dst, dst.Rect, src, src.Bounds().Min, draw.Over)

	switch form {
	case "jpeg", "jpg":
		return jpeg.Encode(output, dst, &jpeg.Options{Quality: q.quality})
	case "png":
		return png.Encode(output, dst)
	case "gif":
		return gif.Encode(output, dst, &gif.Options{NumColors: 256})
	default:
		return fmt.Errorf("unknown image format %s", form)
	}
}

func (q qrim) fatal(form string, err error, args ...any) {
	args = append(args, err)
	fmt.Fprintf(os.Stderr, form+": %s", args...)
	os.Exit(2)
}

func (q qrim) input() (*os.File, error) {
	if q.in == "-" {
		return os.Stdin, nil
	}
	return os.Open(q.in)
}

func (q qrim) output() (*os.File, error) {
	if q.in == "-" {
		return os.Stdout, nil
	}
	return os.Open(q.in)
}

func (q qrim) outputFormat() string {
	if q.out != "-" {
		return filepath.Ext(q.out)
	}
	return q.form
}

func (q qrim) resize() {
	in, err := q.input()
	if err != nil {
		q.fatal("cannot read input", err)
	}
	defer in.Close()
	out, err := q.output()
	if err != nil {
		q.fatal("cannot read input", err)
	}
	defer out.Close()
	err = q.resizeImage(out, in, q.outputFormat())
	if err != nil {
		q.fatal("cannot resize image", err)
	}
}

func main() {
	q := &qrim{}
	flag.StringVar(&q.in, "i", "-", "Input file, or stdin if '-'")
	flag.StringVar(&q.out, "o", "-", "Output file, or stdout if '-'")
	flag.StringVar(&q.target, "t", "", "Target file, or none if not given")
	flag.StringVar(&q.mask, "m", "", "Mask file, or none if not given")
	flag.StringVar(&q.form, "f", "jpeg", "Output format")
	flag.IntVar(&q.width, "W", 240, "Output width")
	flag.IntVar(&q.height, "H", 240, "Output height")
	flag.IntVar(&q.quality, "Q", 90, "Output quality for JPEG")
	flag.Parse()
	q.resize()
}
