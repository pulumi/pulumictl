package cover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"golang.org/x/tools/cover"
)

type mode string

func (m mode) merge(dest, src cover.ProfileBlock) cover.ProfileBlock {
	if m == "set" {
		if src.Count != 0 {
			dest.Count = src.Count
		}
	} else {
		dest.Count += src.Count
	}
	return dest
}

type blockLocation struct {
	startLine, startCol, endLine, endCol int
}

func (l blockLocation) less(m blockLocation) bool {
	return l.startLine < m.startLine || l.startCol < m.startCol || l.endLine < m.endLine || l.endCol < m.endCol
}

type profile struct {
	fileName string
	blocks   map[blockLocation]cover.ProfileBlock
}

func (p *profile) merge(other *cover.Profile, mode mode) {
	for _, block := range other.Blocks {
		loc := blockLocation{block.StartLine, block.StartCol, block.EndLine, block.EndCol}
		if b, ok := p.blocks[loc]; ok {
			p.blocks[loc] = mode.merge(b, block)
		} else {
			p.blocks[loc] = block
		}
	}
}

func (p *profile) write(w io.Writer) error {
	locs := make([]blockLocation, 0, len(p.blocks))
	for loc := range p.blocks {
		locs = append(locs, loc)
	}
	sort.Slice(locs, func(i, j int) bool { return locs[i].less(locs[j]) })

	for _, l := range locs {
		b := p.blocks[l]
		if _, err := fmt.Fprintf(w, "%v:%v.%v,%v.%v %v %v\n",
			p.fileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmt, b.Count); err != nil {
			return err
		}
	}
	return nil
}

type profiles struct {
	mode  mode
	files map[string]*profile
}

func (ps *profiles) merge(other []*cover.Profile) error {
	for _, o := range other {
		switch {
		case ps.mode == "":
			ps.mode = mode(o.Mode)
		case o.Mode != string(ps.mode):
			return fmt.Errorf("%v's coverage mode '%v' does not match the merged mode '%v'", o.FileName, o.Mode, ps.mode)
		}

		p, ok := ps.files[o.FileName]
		if !ok {
			p = &profile{fileName: o.FileName, blocks: map[blockLocation]cover.ProfileBlock{}}
			ps.files[o.FileName] = p
		}
		p.merge(o, ps.mode)
	}
	return nil
}

func (ps *profiles) write(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "mode: %v\n", ps.mode); err != nil {
		return err
	}

	files := make([]string, 0, len(ps.files))
	for fileName := range ps.files {
		files = append(files, fileName)
	}
	sort.Strings(files)

	for _, fileName := range files {
		if err := ps.files[fileName].write(w); err != nil {
			return err
		}
	}
	return nil
}

func coverCommand() *cobra.Command {
	var inPath string
	var outPath string

	command := &cobra.Command{
		Use:   "merge",
		Short: "Merge coverage profiles",
		Long: "Merge coverage profiles.\n" +
			"\n" +
			"merge merges all coverage profiles contained in a given input directory into a\n" +
			"single coverage profile. Coverage profile filenames must end with '.cov'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inPath == "" {
				return fmt.Errorf("the -in flag is required")
			}

			profiles := profiles{files: map[string]*profile{}}

			entries, err := os.ReadDir(inPath)
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == ".cov" {
					path := filepath.Join(inPath, e.Name())
					rawProfiles, err := cover.ParseProfiles(path)
					if err != nil {
						return fmt.Errorf("parsing profiles from '%v': %w", path, err)
					}
					if err = profiles.merge(rawProfiles); err != nil {
						return fmt.Errorf("merging coverage from '%v': %w", path, err)
					}
				}
			}

			outFile := os.Stdout
			if outPath != "" {
				outFile, err = os.Create(outPath)
				if err != nil {
					return fmt.Errorf("creating output file '%v': %w", outPath, err)
				}
				defer outFile.Close()
			}

			if err = profiles.write(outFile); err != nil {
				return fmt.Errorf("writing merged profile: %w", err)
			}

			return nil
		},
	}

	command.PersistentFlags().StringVarP(&inPath,
		"in", "i", "", "the path to the directory containing coverage data to merge")
	command.PersistentFlags().StringVarP(&outPath, "out", "o", "", "the path to the output file")

	return command
}
