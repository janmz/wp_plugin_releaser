package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sshcommands "github.com/janmz/ssh-commands"
	"golang.org/x/crypto/ssh"
)

type sshLog struct{}

func (sshLog) Info(format string, args ...interface{}) {
	logAndPrint(fmt.Sprintf(format, args...))
}

func (sshLog) Warn(format string, args ...interface{}) {
	logAndPrint(fmt.Sprintf(format, args...))
}

func configToSSHOpts(config *ConfigType) (*sshcommands.Opts, error) {
	port := 22
	if config.SSHPort != "" {
		p, err := strconv.Atoi(config.SSHPort)
		if err != nil {
			return nil, fmt.Errorf("%s", t("error.ssh_port_invalid", config.SSHPort))
		}
		port = p
	}

	opts := &sshcommands.Opts{
		Host: config.SSHHost,
		Port: port,
		User: config.SSHUser,
	}

	if config.SSHKeyFile != "" {
		keyPath := config.SSHKeyFile
		key, err := os.ReadFile(keyPath)
		if err != nil {
			logAndPrint(t("log.ssh_key_warning", err))
		} else if _, err := ssh.ParsePrivateKey(key); err != nil {
			logAndPrint(t("log.ssh_key_parse_warning", err))
		} else {
			opts.KeyFile = keyPath
			logAndPrint(t("log.ssh_key_added"))
		}
	}

	if config.SSHPassword != "" {
		opts.Password = config.SSHPassword
		logAndPrint(t("log.ssh_password_added"))
	}

	if opts.KeyFile == "" && opts.Password == "" {
		return nil, fmt.Errorf("%s", t("error.ssh_no_auth"))
	}
	return opts, nil
}

func resolveKnownHostsPath(config *ConfigType, workDir string) string {
	if config.SSHKnownHosts != "" {
		p := config.SSHKnownHosts
		if !filepath.IsAbs(p) {
			p = filepath.Join(workDir, p)
		}
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

func uploadFiles(config *ConfigType, zipPath, updateInfoPath string, workDir string, updateInfo *UpdateInfo, fetchHostKey bool) error {
	logAndPrint(t("log.ssh_upload_start"))

	opts, err := configToSSHOpts(config)
	if err != nil {
		return err
	}

	knownHostsPath := resolveKnownHostsPath(config, workDir)
	if knownHostsPath == "" {
		return fmt.Errorf("%s", t("error.ssh_known_hosts_path"))
	}

	port := opts.Port
	if port <= 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", opts.Host, port)
	logAndPrint(t("log.ssh_connecting", addr))

	log := sshLog{}
	client, err := sshcommands.DialKnownHosts(opts, sshcommands.KnownHostsOptions{
		Path:            knownHostsPath,
		FetchHostKey:    fetchHostKey,
		TrustOnMismatch: fetchHostKey,
	}, log)
	if err != nil {
		if !fetchHostKey && strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("%s", t("error.ssh_host_key_required"))
		}
		return fmt.Errorf(t("error.ssh_connection"), err)
	}
	defer client.Close()
	logAndPrint(t("log.ssh_connected"))

	remoteLocalPath, err := parseRemotePath(updateInfo.DownloadURL, config.SSHDirBase)
	if err != nil {
		return err
	}
	logAndPrint(t("log.remote_path", remoteLocalPath))

	if err := sshcommands.MkdirAllRemote(client, remoteLocalPath, log); err != nil {
		logAndPrint(t("log.remote_dir_warning", err))
	}

	if err := sshcommands.UploadFileIfNewer(client, zipPath, filepath.Join(remoteLocalPath, filepath.Base(zipPath)), log); err != nil {
		return fmt.Errorf(t("error.zip_upload"), err)
	}

	if err := sshcommands.UploadFileIfNewer(client, updateInfoPath, filepath.Join(remoteLocalPath, "update_info.json"), log); err != nil {
		return fmt.Errorf(t("error.update_info_upload"), err)
	}

	updatePath := filepath.Join(workDir, "Updates")
	if len(updateInfo.Banners) > 0 {
		for key, bannerURL := range updateInfo.Banners {
			if _, err := url.Parse(bannerURL); err == nil {
				bannerFilename := filepath.Base(bannerURL)
				localBannerPath := filepath.Join(updatePath, bannerFilename)
				if _, err := os.Stat(localBannerPath); os.IsNotExist(err) {
					logAndPrint(t("log.banner_not_found", key, localBannerPath))
				} else {
					remoteBannerPath := filepath.Join(remoteLocalPath, bannerFilename)
					if err := sshcommands.UploadFileIfNewer(client, localBannerPath, remoteBannerPath, log); err != nil {
						return fmt.Errorf(t("error.banner_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.banner_no_url", key, redactSensitiveURL(bannerURL)))
			}
		}
	}
	if len(updateInfo.Icons) > 0 {
		for key, iconURL := range updateInfo.Icons {
			if _, err := url.Parse(iconURL); err == nil {
				iconFilename := filepath.Base(iconURL)
				localIconPath := filepath.Join(updatePath, iconFilename)
				if _, err := os.Stat(localIconPath); os.IsNotExist(err) {
					logAndPrint(t("log.icon_not_found", key, localIconPath))
				} else {
					remoteIconPath := filepath.Join(remoteLocalPath, iconFilename)
					if err := sshcommands.UploadFileIfNewer(client, localIconPath, remoteIconPath, log); err != nil {
						return fmt.Errorf(t("error.icon_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.icon_no_url", key, redactSensitiveURL(iconURL)))
			}
		}
	}

	return nil
}

func parseRemotePath(downloadURL string, basedir string) (string, error) {
	urlInfo, err := url.Parse(downloadURL)
	if err != nil {
		return "", err
	}
	p := urlInfo.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if strings.HasSuffix(p, "/") {
		return "", fmt.Errorf("%s", t("error.url_ends_directory", downloadURL))
	}
	pos := strings.LastIndex(p, "/")
	if pos < 0 {
		return "", fmt.Errorf("%s", t("error.url_no_filename", downloadURL))
	}
	p = p[:pos]

	basedir = strings.TrimSuffix(basedir, "/")
	return basedir + p, nil
}
