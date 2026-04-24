package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	urfave "github.com/urfave/cli/v2"

	"github.com/zbum/nexus3-cli/internal/config"
	"github.com/zbum/nexus3-cli/internal/registry"
)

// componentUploadTime returns the most recent blob-creation time across the
// component's assets, falling back to LastModified when BlobCreated is zero.
// For a Docker tag this is close to "when the tag was (re)pushed".
func componentUploadTime(c registry.Component) time.Time {
	var latest time.Time
	for _, a := range c.Assets {
		t := a.BlobCreated
		if t.IsZero() {
			t = a.LastModified
		}
		if t.After(latest) {
			latest = t
		}
	}
	return latest
}

func imageCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "image",
		Usage: "Manage Docker images stored in Nexus",
		Subcommands: []*urfave.Command{
			imageListCommand(),
			imageTagsCommand(),
			imageInfoCommand(),
			imageDeleteCommand(),
			imageSizeCommand(),
		},
	}
}

func newClient() (*registry.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Repository == "" {
		return nil, fmt.Errorf("nexus_repository not set in config; run `nexus3-cli configure`")
	}
	return registry.New(cfg.Host, cfg.Repository, cfg.Username, cfg.Password), nil
}

func imageListCommand() *urfave.Command {
	return &urfave.Command{
		Name:    "ls",
		Aliases: []string{"list"},
		Usage:   "List all Docker images in the configured repository",
		Action: func(c *urfave.Context) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			images, err := client.ListImages()
			if err != nil {
				return err
			}
			sort.Strings(images)
			for _, name := range images {
				fmt.Fprintln(c.App.Writer, name)
			}
			fmt.Fprintf(c.App.Writer, "\n%s — %d image(s)\n", client.Repository(), len(images))
			return nil
		},
	}
}

func imageTagsCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "tags",
		Usage: "List tags for a Docker image",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "image name (e.g. my-app)", Required: true},
		},
		Action: func(c *urfave.Context) error {
			name := c.String("name")
			client, err := newClient()
			if err != nil {
				return err
			}
			tags, err := client.ListTags(name)
			if err != nil {
				return err
			}
			sort.Sort(naturalSort(tags))
			for _, t := range tags {
				fmt.Fprintln(c.App.Writer, t)
			}
			fmt.Fprintf(c.App.Writer, "\n%s — %d tag(s)\n", name, len(tags))
			return nil
		},
	}
}

func imageInfoCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "info",
		Usage: "Show details for an image:tag",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "image name", Required: true},
			&urfave.StringFlag{Name: "tag", Aliases: []string{"t"}, Usage: "image tag", Required: true},
		},
		Action: func(c *urfave.Context) error {
			name, tag := c.String("name"), c.String("tag")
			client, err := newClient()
			if err != nil {
				return err
			}
			comp, err := client.GetComponent(name, tag)
			if err != nil {
				return err
			}

			var total int64
			for _, a := range comp.Assets {
				total += a.FileSize
			}

			w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Image:\t%s:%s\n", comp.Name, comp.Version)
			fmt.Fprintf(w, "Repository:\t%s\n", comp.Repository)
			fmt.Fprintf(w, "Format:\t%s\n", comp.Format)
			fmt.Fprintf(w, "Component ID:\t%s\n", comp.ID)
			fmt.Fprintf(w, "Assets:\t%d\n", len(comp.Assets))
			fmt.Fprintf(w, "Total size:\t%s\n", humanBytes(total))
			w.Flush()

			fmt.Fprintln(c.App.Writer)
			aw := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
			fmt.Fprintln(aw, "PATH\tSIZE\tCONTENT-TYPE\tLAST MODIFIED")
			for _, a := range comp.Assets {
				lm := ""
				if !a.LastModified.IsZero() {
					lm = a.LastModified.Format("2006-01-02 15:04:05")
				}
				fmt.Fprintf(aw, "%s\t%s\t%s\t%s\n", a.Path, humanBytes(a.FileSize), a.ContentType, lm)
			}
			return aw.Flush()
		},
	}
}

// parseDuration parses a human-friendly duration string.
// Supports Go's time.ParseDuration syntax (e.g. "720h") plus a "d" suffix
// for days (e.g. "30d" = 720h).
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		s = strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(s, "%d", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid duration %q: days must be a positive integer", s+"d")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: use e.g. 30d, 720h", s)
	}
	return d, nil
}

func imageDeleteCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "delete",
		Usage: "Delete a specific tag, or keep only the N most recent tags",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "image name", Required: true},
			&urfave.StringFlag{Name: "tag", Aliases: []string{"t"}, Usage: "tag to delete"},
			&urfave.IntFlag{Name: "keep", Aliases: []string{"k"}, Usage: "keep only the N most recently uploaded tags (by blob creation time)"},
			&urfave.StringFlag{Name: "keep-within", Usage: "never delete tags uploaded within this duration (e.g. 30d, 720h)"},
			&urfave.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "skip confirmation prompt"},
		},
		Action: func(c *urfave.Context) error {
			name := c.String("name")
			tag := c.String("tag")
			keep := c.Int("keep")
			keepWithin := c.String("keep-within")
			if tag == "" && keep <= 0 {
				return fmt.Errorf("either --tag or --keep must be provided")
			}
			if tag != "" && keep > 0 {
				return fmt.Errorf("--tag and --keep are mutually exclusive")
			}
			if tag != "" && keepWithin != "" {
				return fmt.Errorf("--tag and --keep-within are mutually exclusive")
			}

			var keepWithinDur time.Duration
			if keepWithin != "" {
				d, err := parseDuration(keepWithin)
				if err != nil {
					return err
				}
				keepWithinDur = d
			}

			client, err := newClient()
			if err != nil {
				return err
			}

			comps, err := client.ListComponentsByName(name)
			if err != nil {
				return err
			}
			if len(comps) == 0 {
				return fmt.Errorf("no tags found for image %q", name)
			}

			var targets []registry.Component
			if tag != "" {
				for _, comp := range comps {
					if comp.Version == tag {
						targets = append(targets, comp)
						break
					}
				}
				if len(targets) == 0 {
					return fmt.Errorf("tag %s:%s not found", name, tag)
				}
			} else {
				sort.SliceStable(comps, func(i, j int) bool {
					ti, tj := componentUploadTime(comps[i]), componentUploadTime(comps[j])
					if !ti.Equal(tj) {
						return ti.Before(tj) // oldest first
					}
					return naturalLess(comps[i].Version, comps[j].Version)
				})
				if len(comps) <= keep {
					fmt.Fprintf(c.App.Writer, "Nothing to delete: %d tag(s) found, keeping %d\n", len(comps), keep)
					return nil
				}
				candidates := comps[:len(comps)-keep]

				if keepWithinDur > 0 {
					cutoff := time.Now().Add(-keepWithinDur)
					for _, comp := range candidates {
						if componentUploadTime(comp).Before(cutoff) {
							targets = append(targets, comp)
						}
					}
				} else {
					targets = candidates
				}

				if len(targets) == 0 {
					fmt.Fprintf(c.App.Writer, "Nothing to delete: all candidates are within the --keep-within window\n")
					return nil
				}
			}

			if !c.Bool("yes") {
				fmt.Fprintf(c.App.Writer, "About to delete %d tag(s) from %s (oldest uploads first):\n", len(targets), name)
				tw := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
				for _, t := range targets {
					ts := componentUploadTime(t)
					when := "—"
					if !ts.IsZero() {
						when = ts.Format("2006-01-02 15:04:05")
					}
					fmt.Fprintf(tw, "  - %s\t(uploaded %s)\n", t.Version, when)
				}
				tw.Flush()
				fmt.Fprint(c.App.Writer, "Proceed? [y/N]: ")
				var ans string
				fmt.Scanln(&ans)
				ans = strings.TrimSpace(strings.ToLower(ans))
				if ans != "y" && ans != "yes" {
					fmt.Fprintln(c.App.Writer, "Aborted.")
					return nil
				}
			}

			var failed int
			for _, t := range targets {
				if err := client.DeleteComponent(t.ID); err != nil {
					fmt.Fprintf(c.App.ErrWriter, "%s:%s — delete failed: %v\n", name, t.Version, err)
					failed++
					continue
				}
				fmt.Fprintf(c.App.Writer, "deleted %s:%s (id=%s)\n", name, t.Version, t.ID)
			}
			if failed > 0 {
				return fmt.Errorf("%d tag(s) failed to delete", failed)
			}
			fmt.Fprintln(c.App.Writer, "\nNote: run the Nexus `Admin - Compact blob store` task to reclaim disk space.")
			return nil
		},
	}
}

func imageSizeCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "size",
		Usage: "Show total size of an image (sum over all tags)",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "image name", Required: true},
		},
		Action: func(c *urfave.Context) error {
			name := c.String("name")
			client, err := newClient()
			if err != nil {
				return err
			}
			comps, err := client.ListComponentsByName(name)
			if err != nil {
				return err
			}
			sort.Slice(comps, func(i, j int) bool {
				return naturalLess(comps[i].Version, comps[j].Version)
			})

			w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TAG\tASSETS\tSIZE\tCOMPONENT ID")
			var total int64
			for _, comp := range comps {
				var sz int64
				for _, a := range comp.Assets {
					sz += a.FileSize
				}
				total += sz
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", comp.Version, len(comp.Assets), humanBytes(sz), comp.ID)
			}
			w.Flush()
			fmt.Fprintf(c.App.Writer, "\n%s — %d tag(s), %s total (shared layers counted per tag)\n", name, len(comps), humanBytes(total))
			return nil
		},
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
