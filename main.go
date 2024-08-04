package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"strconv"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

const (
	uriX8664        = "https://github.com/cbeuw/Cloak/releases/download/v2.9.0/ck-client-linux-amd64-v2.9.0"
	uriArm64        = "https://github.com/cbeuw/Cloak/releases/download/v2.9.0/ck-client-linux-arm64-v2.9.0"
	cloakSystemPath = "/usr/bin/ck-client"
)

type Config struct {
	Clients map[string]ClientConfig
}

type ClientConfig struct {
	Server string `yaml:"server"`
	Port   int    `yaml:"port"`
	Listen int    `yaml:"listen"`
	Config string `yaml:"config"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	if !checkCloak() {
		dieOnError("Cloak install failed.", installCloak())
	}

	dieOnError("starting config file watcher failed.", startWatcher(ctx))

}

func checkCloak() bool {
	_, err := exec.LookPath("ck-client")

	_, staterr := os.Stat(cloakSystemPath)

	return err == nil || !errors.Is(staterr, os.ErrNotExist)
}

func getCloak() string {
	str, _ := exec.LookPath("ck-client")
	if len(str) == 0 {
		return cloakSystemPath
	}

	return str
}

func installCloak() error {
	arch := runtime.GOARCH
	var cloakURI string

	fmt.Printf("Installing Cloak for %s\n", arch)

	switch arch {
	case "amd64":
		cloakURI = uriX8664
	case "arm64":
		cloakURI = uriArm64
	default:
		fmt.Printf("Unsupported architecture: %s\n", arch)
		return fmt.Errorf("unsupported arch")
	}

	err := downloadFile(cloakSystemPath, cloakURI)
	if err != nil {
		fmt.Printf("Failed to download Cloak: %v\n", err)
		return err
	} else {
		fmt.Println("Cloak installed successfully")
	}

	return os.Chmod(cloakSystemPath, 0667)
}

func startWatcher(ctx context.Context) error {
	usr, err := user.Current()

	if err != nil {
		return err
	}

	configRoot := filepath.Join(usr.HomeDir, ".config", "cloak")

	fmt.Println("Starting cloak")

	fmt.Println(configRoot)
	os.MkdirAll(configRoot, 0667)
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	defer watcher.Close()

	configFile := filepath.Join(configRoot, "config.yml")

	fmt.Println(configFile)
	if err := watcher.Add(configFile); err != nil {
		return err
	}

	cctx, cancel := context.WithCancel(ctx)

	defer cancel()

	var clean func()

	if clean, err = startCloak(cctx, configRoot); err != nil {
		return err
	}

	for {
		select {
		case event := <-watcher.Events:
			fmt.Println(event)
			if event.Op.Has(fsnotify.Write) {
				time.Sleep(time.Second)
				cancel()

				if clean != nil {
					clean()
				}

				time.Sleep(time.Second * 2)

				cctx, cancel = context.WithCancel(ctx)

				if clean, err = startCloak(cctx, configRoot); err != nil {
					return err
				}
			}
		case err := <-watcher.Errors:
			fmt.Println(err)
		}
	}
}

func startCloak(ctx context.Context, configRoot string) (func(), error) {
	config, err := prepareConfig(configRoot)

	if err != nil {
		return nil, err
	}

	var list []func()

	for k, c := range config.Clients {
		fName := filepath.Join(configRoot, fmt.Sprintf(".config-%s.json", k))
		logFName := filepath.Join(configRoot, fmt.Sprintf(".log-%s.log", k))

		if err := os.WriteFile(fName, []byte(c.Config), 0666); err != nil {
			return nil, err
		}

		file, err := os.Create(logFName)

		if err != nil {
			return nil, err
		}

		cmd := exec.CommandContext(ctx, getCloak(), "-c", fName, "-s", c.Server, "-p", strconv.Itoa(c.Port), "-l", strconv.Itoa(c.Listen))

		// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdout = file
		cmd.Stderr = file

		list = append(list, func() {
			fmt.Println("killing ck-client")
			cmd.Process.Kill()
			file.Close()
		})

		if err := cmd.Start(); err != nil {
			return nil, err
		}

		fmt.Printf("Started config %s\n", fName)
	}

	go func() {
		<-ctx.Done()
		for _, l := range list {
			l()
		}
	}()

	return func() {
		for _, l := range list {
			l()
		}
	}, nil
}

func prepareConfig(configRoot string) (*Config, error) {
	fmt.Println("Config file changed. refreshing.")

	// Read the config
	file, err := os.Open(filepath.Join(configRoot, "config.yml"))

	if err != nil {
		fmt.Printf("Failed to open config.yml %v", err)
		return nil, err
	}

	arr, _ := io.ReadAll(file)

	config := &Config{}

	if err := yaml.Unmarshal(arr, &config); err != nil {
		fmt.Printf("Failed to open config.yml %v", err)
		return nil, err
	}

	fmt.Println(config)
	return config, nil
}

func downloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func dieOnError(reason string, err error) {
	if err != nil {
		fmt.Printf("%s %s", reason, err)
		panic(err)
	}
}
