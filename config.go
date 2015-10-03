package express

import (
	"bufio"
	"fmt"
	"github.com/olebedev/config"
	"os"
	"regexp"
	"path/filepath"
)

var (
	ymlfile = regexp.MustCompile(`(.*)\.yml$`)
	regex = regexp.MustCompile(`(.*)\$([^\n\t\s]+)`)
)

func parse(e Environment, file string) (*config.Config, error) {

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	reader := bufio.NewReader(f)
	scanner := bufio.NewScanner(reader)

	data := make([]byte, 0)

	for scanner.Scan() {
		parsed := scanner.Text() + "\n"
		p := regex.FindAllStringSubmatch(parsed, -1)
		if len(p) > 0 {
			parsed = fmt.Sprintf("%s\"%s\"\n", p[0][1], os.Getenv(p[0][2]))
		}

		data = append(data, []byte(parsed)...)
	}

	cfg, err := config.ParseYaml(string(data))
	if err != nil {
		return nil, err
	}

	env, err := cfg.Get(e.String())
	if err == nil {

		def, err := cfg.Get("default")

		if err == nil {
			return def.Extend(env)
		}

		return env, nil
	}

	return nil, nil
}

func scan(e Environment, root string) (cfg *config.Config, err error) {
	cfg = &config.Config{Root: map[string]interface{}{}}
	err = filepath.Walk(root, func(p string, info os.FileInfo, err error) (error) {
		if err != nil {
			return err
		}

		if !info.IsDir() && ymlfile.MatchString(info.Name()) {

			k := ymlfile.FindAllStringSubmatch(info.Name(), -1)

			if c, err := parse(e, p); err == nil {
				if c != nil {
					cfg.Set(k[0][1], c.Root)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func NewConfig(e Environment, p string) (c *config.Config, err error) {
	return scan(e, p)
}