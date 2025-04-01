package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"
)

var (
	format     string
	outputFile string
	header     string
)

func init() {
	flag.StringVar(&format, "format", "yaml", `output format. May be "json", "yaml" or "go"`)
	flag.StringVar(&outputFile, "out", "", "output file name")
	flag.StringVar(&header, "header", "", "path to an optional header text file, for go format")
	flag.Parse()

	switch format {
	case "json", "yaml", "go":
	default:
		fmt.Fprintln(os.Stderr, "format must be one of [json, yaml, go]")
		os.Exit(1)
	}
}

func main() {
	cr := components.GetOperatorCR()

	out := os.Stdout
	if outputFile != "" {
		var err error
		out, err = os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't create output file %s; %v\n", outputFile, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	switch format {
	case "json":
		if err := writeJSON(cr, out); err != nil {
			fmt.Fprintf(os.Stderr, "can't write json file; %v", err)
			os.Exit(1)
		}

	case "yaml":
		if err := util.MarshallObject(cr, out); err != nil {
			fmt.Fprintf(os.Stderr, "can't write yaml file; %v", err)
			os.Exit(1)
		}

	case "go":
		if err := generateGo(out, cr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func writeJSON(cr *v1beta1.HyperConverged, w io.Writer) error {
	dec := json.NewEncoder(w)
	dec.SetIndent("", "  ")
	return dec.Encode(cr)
}

func generateGo(w io.Writer, cr *v1beta1.HyperConverged) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't get current working directory; %v", err)
	}

	_, pkg := path.Split(wd)
	if _, err = fmt.Fprintln(w, "// Code generated by tools/crwriter; DO NOT EDIT."); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(w, "package", pkg); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(w); err != nil {
		return err
	}

	if len(header) > 0 {
		if err = writeBoilerplate(w); err != nil {
			return fmt.Errorf("can't write boilerplate; %v", err)
		}

		if _, err = fmt.Fprintln(w); err != nil {
			return err
		}
	}

	if _, err = fmt.Fprint(w, "var hyperConvergedCRDefault = []byte(`"); err != nil {
		return err
	}
	err = writeJSON(cr, w)
	if err != nil {
		return fmt.Errorf("can't write json output; %v", err)
	}
	if _, err = fmt.Fprintln(w, "`)"); err != nil {
		return fmt.Errorf("can't write json output; %v", err)
	}

	return nil
}

func writeBoilerplate(w io.Writer) error {
	boilerplate, err := os.ReadFile(header)
	if err != nil {
		return fmt.Errorf("can't read boilerplate; %v", err)
	}

	year := []byte(time.Now().UTC().Format("2006"))

	boilerplate = bytes.ReplaceAll(boilerplate, []byte("YEAR"), year)
	_, err = w.Write(boilerplate)

	return err
}
