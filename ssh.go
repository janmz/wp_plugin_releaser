package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func uploadFiles(config *ConfigType, zipPath, updateInfoPath string, workDir string, updateInfo *UpdateInfo, trustServer bool) error {
	logAndPrint(t("log.ssh_upload_start"))

	var authMethods []ssh.AuthMethod

	if config.SSHKeyFile != "" {
		key, err := os.ReadFile(config.SSHKeyFile)
		if err != nil {
			logAndPrint(t("log.ssh_key_warning", err))
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				logAndPrint(t("log.ssh_key_parse_warning", err))
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				logAndPrint(t("log.ssh_key_added"))
			}
		}
	}

	if config.SSHPassword != "" {
		authMethods = append(authMethods, ssh.Password(config.SSHPassword))
		logAndPrint(t("log.ssh_password_added"))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("%s", t("error.ssh_no_auth"))
	}

	fetchServerHostKeyLine := func(host string, port string, user string, auth []ssh.AuthMethod) (string, error) {
		if host == "" {
			return "", fmt.Errorf("ssh host is empty")
		}
		if port == "" {
			port = "22"
		}
		addr := fmt.Sprintf("%s:%s", host, port)
		var capturedKey ssh.PublicKey
		hostKeyCB := func(_ string, _ net.Addr, key ssh.PublicKey) error {
			capturedKey = key
			return nil
		}
		cfg := &ssh.ClientConfig{
			User:            user,
			Auth:            auth,
			HostKeyCallback: hostKeyCB,
			Timeout:         30 * time.Second,
		}
		client, err := ssh.Dial("tcp", addr, cfg)
		if err != nil {
			return "", err
		}
		defer client.Close()
		if capturedKey == nil {
			return "", fmt.Errorf("server did not send a host key")
		}
		return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(capturedKey))), nil
	}

	ensureKnownHostsHasServerKey := func(knownHostsPath string, host string, port string) {
		if !trustServer {
			return
		}
		if knownHostsPath == "" || host == "" {
			return
		}
		keyLine, err := fetchServerHostKeyLine(host, port, config.SSHUser, authMethods)
		if err != nil {
			logAndPrint(fmt.Sprintf("Could not fetch SSH host key from server: %v", err))
			return
		}
		hostToken := host
		if port != "" && port != "22" {
			hostToken = fmt.Sprintf("[%s]:%s", host, port)
		}
		entry := fmt.Sprintf("%s %s\n", hostToken, keyLine)
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			logAndPrint(fmt.Sprintf("Could not create known_hosts directory: %v", err))
			return
		}
		f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // # nosec G304
		if err != nil {
			logAndPrint(fmt.Sprintf("Could not open known_hosts file: %v", err))
			return
		}
		defer f.Close()
		if _, err := f.WriteString(entry); err != nil {
			logAndPrint(fmt.Sprintf("Could not write known_hosts file: %v", err))
			return
		}
		logAndPrint(fmt.Sprintf("SSH host key appended to known_hosts: %s", knownHostsPath))
	}

	hostKeyCallback := ssh.InsecureIgnoreHostKey() // #nosec G106
	hostKeyVerified := false
	knownHostsPathUsed := ""
	tryKnownHosts := func(path string) bool {
		if path == "" {
			return false
		}
		if _, err := os.Stat(path); err != nil {
			return false
		}
		cb, err := knownhosts.New(path)
		if err != nil {
			logAndPrint(fmt.Sprintf("known_hosts file invalid: %v", err))
			return false
		}
		hostKeyCallback = cb
		hostKeyVerified = true
		knownHostsPathUsed = path
		logAndPrint(fmt.Sprintf("SSH host key verification enabled: %s", path))
		return true
	}

	if config.SSHKnownHosts != "" {
		khPath := config.SSHKnownHosts
		if !filepath.IsAbs(khPath) {
			khPath = filepath.Join(workDir, khPath)
		}
		ensureKnownHostsHasServerKey(khPath, config.SSHHost, config.SSHPort)
		tryKnownHosts(khPath)
	}
	if !hostKeyVerified {
		if home, err := os.UserHomeDir(); err == nil {
			defaultKH := filepath.Join(home, ".ssh", "known_hosts")
			ensureKnownHostsHasServerKey(defaultKH, config.SSHHost, config.SSHPort)
			tryKnownHosts(defaultKH)
		}
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	port := config.SSHPort
	if port == "" {
		port = "22"
	}

	addr := fmt.Sprintf("%s:%s", config.SSHHost, port)
	logAndPrint(t("log.ssh_connecting", addr))

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		if trustServer && strings.Contains(err.Error(), "knownhosts: key mismatch") && knownHostsPathUsed != "" {
			logAndPrint("SSH host key mismatch detected; trusting server and updating known_hosts")
			ensureKnownHostsHasServerKey(knownHostsPathUsed, config.SSHHost, port)
			cb, err2 := knownhosts.New(knownHostsPathUsed)
			if err2 == nil {
				sshConfig2 := &ssh.ClientConfig{
					User:            config.SSHUser,
					Auth:            authMethods,
					HostKeyCallback: cb,
					Timeout:         30 * time.Second,
				}
				client2, err2 := ssh.Dial("tcp", addr, sshConfig2)
				if err2 == nil {
					client = client2
					err = nil
				} else {
					err = err2
				}
			}
		}
		if err != nil {
			return fmt.Errorf(t("error.ssh_connection"), err)
		}
	}
	defer client.Close()
	logAndPrint(t("log.ssh_connected"))

	remoteLocalPath, err := parseRemotePath(updateInfo.DownloadURL, config.SSHDirBase)
	if err != nil {
		return err
	}
	logAndPrint(t("log.remote_path", remoteLocalPath))

	err = createRemoteDir(client, remoteLocalPath)
	if err != nil {
		logAndPrint(t("log.remote_dir_warning", err))
	}

	err = uploadFileViaSFTP(client, zipPath, filepath.Join(remoteLocalPath, filepath.Base(zipPath)))
	if err != nil {
		return fmt.Errorf(t("error.zip_upload"), err)
	}

	err = uploadFileViaSFTP(client, updateInfoPath, filepath.Join(remoteLocalPath, "update_info.json"))
	if err != nil {
		return fmt.Errorf(t("error.update_info_upload"), err)
	}

	updatePath := filepath.Join(workDir, "Updates")
	if len(updateInfo.Banners) > 0 {
		for key, bannerUrl := range updateInfo.Banners {
			if _, err := url.Parse(bannerUrl); err == nil {
				bannerFilename := filepath.Base(bannerUrl)
				localBannerPath := filepath.Join(updatePath, bannerFilename)
				if _, err := os.Stat(localBannerPath); os.IsNotExist(err) {
					logAndPrint(t("log.banner_not_found", key, localBannerPath))
				} else {
					remoteBannerPath := filepath.Join(remoteLocalPath, bannerFilename)
					err = uploadFileViaSFTP(client, localBannerPath, remoteBannerPath)
					if err != nil {
						return fmt.Errorf(t("error.banner_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.banner_no_url", key, redactSensitiveURL(bannerUrl)))
			}
		}
	}
	if len(updateInfo.Icons) > 0 {
		for key, iconUrl := range updateInfo.Icons {
			if _, err := url.Parse(iconUrl); err == nil {
				iconFilename := filepath.Base(iconUrl)
				localIconPath := filepath.Join(updatePath, iconFilename)
				if _, err := os.Stat(localIconPath); os.IsNotExist(err) {
					logAndPrint(t("log.icon_not_found", key, localIconPath))
				} else {
					remoteIconPath := filepath.Join(remoteLocalPath, iconFilename)
					err = uploadFileViaSFTP(client, localIconPath, remoteIconPath)
					if err != nil {
						return fmt.Errorf(t("error.icon_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.icon_no_url", key, redactSensitiveURL(iconUrl)))
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

func createRemoteDir(client *ssh.Client, remotePath string) error {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	remotePath = filepath.ToSlash(remotePath)
	remotePath = path.Clean("/" + strings.TrimPrefix(remotePath, "/"))
	if err := sftpClient.MkdirAll(remotePath); err != nil {
		return err
	}

	logAndPrint(t("log.remote_dir_created", remotePath))
	return nil
}

func uploadFileViaSFTP(client *ssh.Client, localPath, remotePath string) error {
	remotePath = filepath.ToSlash(remotePath)
	remotePath = path.Clean("/" + strings.TrimPrefix(remotePath, "/"))
	logAndPrint(t("log.uploading_file", localPath, remotePath))

	localInfo, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	remoteModTime, err := getRemoteFileModTime(client, remotePath)
	if err == nil {
		if !localInfo.ModTime().After(remoteModTime) {
			logAndPrint(t("log.file_already_current", filepath.Base(localPath)))
			return nil
		}
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	localFile, err := os.Open(localPath) // # nosec G304
	if err != nil {
		return err
	}
	defer localFile.Close()

	remoteDir := filepath.Dir(remotePath)
	remoteDir = filepath.ToSlash(remoteDir)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return err
	}

	remoteFile, err := sftpClient.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return err
	}
	if err := sftpClient.Chtimes(remotePath, time.Now(), localInfo.ModTime()); err != nil {
		logAndPrint(fmt.Sprintf("could not preserve remote file timestamp: %v", err))
	}

	logAndPrint(t("log.file_uploaded", filepath.Base(localPath)))
	return nil
}

func getRemoteFileModTime(client *ssh.Client, remotePath string) (time.Time, error) {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return time.Time{}, err
	}
	defer sftpClient.Close()

	remotePath = filepath.ToSlash(remotePath)
	remotePath = path.Clean("/" + strings.TrimPrefix(remotePath, "/"))
	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

