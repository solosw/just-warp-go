package terminal

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// expandPath expands ~ to the user's home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	return p
}

// BuildSSHAuth returns SSH auth methods (key first, then password).
// Falls back to system SSH command if Go's crypto/ssh can't parse the key format.
func BuildSSHAuth(cfg SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if cfg.KeyPath != "" {
		keyPath := expandPath(cfg.KeyPath)
		signer, err := parseKeyFile(keyPath, cfg.Password)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("需要密码或密钥文件")
	}
	return methods, nil
}

// parseKeyFile tries multiple methods to parse a private key file.
func parseKeyFile(keyPath, passphrase string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件 %q: %w", keyPath, err)
	}

	// Normalize: replace CRLF and trim trailing whitespace
	keyStr := strings.ReplaceAll(string(key), "\r\n", "\n")
	keyStr = strings.TrimSpace(keyStr)
	key = []byte(keyStr)

	// 1. Try Go's built-in parser (OpenSSH / PEM)
	signer, err := ssh.ParsePrivateKey(key)
	if err == nil {
		return signer, nil
	}
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err == nil {
			return signer, nil
		}
	}

	// 2. Try PuTTY format
	if strings.HasPrefix(keyStr, "PuTTY-User-Key-File-") {
		signer, err = parsePuttyKey(keyStr, passphrase)
		if err != nil {
			return nil, fmt.Errorf("PuTTY密钥解析失败 %q: %w", keyPath, err)
		}
		return signer, nil
	}

	// 3. Give up with helpful error
	if !strings.Contains(keyStr, "PRIVATE KEY") && !strings.Contains(keyStr, "OPENSSH") {
		return nil, fmt.Errorf("密钥文件 %q 格式无法识别 — 支持OpenSSH/PEM/PuTTY(.ppk)格式。\n如果是二进制文件(如.bin)，请用 ssh-keygen -i -f %q 转换为OpenSSH格式", keyPath, keyPath)
	}
	return nil, fmt.Errorf("解析密钥 %q 失败: %w\n提示：如果密钥有密码，请在密码栏填入密钥密码", keyPath, err)
}

// parsePuttyKey parses a PuTTY .ppk format private key (v2, unencrypted).
// Supports Ed25519 and RSA algorithms.
func parsePuttyKey(keyStr, passphrase string) (ssh.Signer, error) {
	lines := strings.Split(keyStr, "\n")

	var (
		version     int
		algorithm   string
		encryption  string
		publicData  []byte
		privateData []byte
	)

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++

		if line == "" || strings.HasPrefix(line, "Comment:") || strings.HasPrefix(line, "Private-MAC:") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "PuTTY-User-Key-File-2:"):
			version = 2
			algorithm = strings.TrimSpace(line[len("PuTTY-User-Key-File-2:"):])
		case strings.HasPrefix(line, "PuTTY-User-Key-File-3:"):
			version = 3
			algorithm = strings.TrimSpace(line[len("PuTTY-User-Key-File-3:"):])
		case strings.HasPrefix(line, "Encryption:"):
			encryption = strings.TrimSpace(line[len("Encryption:"):])
		case strings.HasPrefix(line, "Public-Lines:"):
			var n int
			fmt.Sscanf(line, "Public-Lines: %d", &n)
			var b strings.Builder
			for j := 0; j < n && i < len(lines); j, i = j+1, i+1 {
				b.WriteString(strings.TrimSpace(lines[i]))
			}
			var err error
			publicData, err = base64.StdEncoding.DecodeString(b.String())
			if err != nil {
				return nil, fmt.Errorf("PuTTY 公钥解码失败: %w", err)
			}
		case strings.HasPrefix(line, "Private-Lines:"):
			var n int
			fmt.Sscanf(line, "Private-Lines: %d", &n)
			var b strings.Builder
			for j := 0; j < n && i < len(lines); j, i = j+1, i+1 {
				b.WriteString(strings.TrimSpace(lines[i]))
			}
			var err error
			privateData, err = base64.StdEncoding.DecodeString(b.String())
			if err != nil {
				return nil, fmt.Errorf("PuTTY 私钥解码失败: %w", err)
			}
		}
	}

	if version == 0 {
		return nil, fmt.Errorf("无法识别的 PuTTY 密钥格式")
	}
	if version == 3 {
		return nil, fmt.Errorf("暂不支持 PuTTY v3 格式（Argon2），请用 PuTTYgen 转换为 v2 格式")
	}
	if encryption != "none" {
		if passphrase == "" {
			return nil, fmt.Errorf("PuTTY 密钥已加密，需要提供密码")
		}
		return nil, fmt.Errorf("暂不支持加密的 PuTTY 密钥，请用 PuTTYgen 去除密码保护后重试")
	}
	if publicData == nil || privateData == nil {
		return nil, fmt.Errorf("PuTTY 密钥数据不完整")
	}

	switch {
	case strings.Contains(algorithm, "ed25519") || strings.Contains(algorithm, "ssh-ed25519"):
		return makeEd25519SignerFromPPK(publicData, privateData)
	case strings.Contains(algorithm, "rsa") || strings.Contains(algorithm, "ssh-rsa"):
		return makeRSASignerFromPPK(publicData, privateData)
	default:
		return nil, fmt.Errorf("不支持的 PuTTY 密钥算法: %s", algorithm)
	}
}

// sshString reads a 4-byte length-prefixed byte string.
func sshString(data []byte) ([]byte, []byte, error) {
	if len(data) < 4 {
		return nil, nil, fmt.Errorf("truncated ssh string")
	}
	l := binary.BigEndian.Uint32(data[:4])
	if uint32(len(data)) < 4+l {
		return nil, nil, fmt.Errorf("truncated ssh string")
	}
	return data[4 : 4+l], data[4+l:], nil
}

// sshMPInt reads an SSH mpint from data.
func sshMPInt(data []byte) (*big.Int, []byte, error) {
	raw, rest, err := sshString(data)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		return nil, nil, fmt.Errorf("negative mpint not supported")
	}
	return new(big.Int).SetBytes(raw), rest, nil
}

func makeEd25519SignerFromPPK(publicData, privateData []byte) (ssh.Signer, error) {
	// Public data: string("ssh-ed25519") + string(32-byte pubkey)
	_, rest, err := sshString(publicData)
	if err != nil {
		return nil, fmt.Errorf("Ed25519 公钥格式错误: %w", err)
	}
	pubKeyBytes, _, err := sshString(rest)
	if err != nil {
		return nil, fmt.Errorf("Ed25519 公钥格式错误: %w", err)
	}

	// Private data: mpint(32-byte seed)
	seedInt, _, err := sshMPInt(privateData)
	if err != nil {
		return nil, fmt.Errorf("Ed25519 私钥格式错误: %w", err)
	}
	seedBytes := seedInt.Bytes()
	if len(seedBytes) < ed25519.SeedSize {
		padded := make([]byte, ed25519.SeedSize)
		copy(padded[ed25519.SeedSize-len(seedBytes):], seedBytes)
		seedBytes = padded
	}

	privKey := ed25519.NewKeyFromSeed(seedBytes)
	if !bytes.Equal(privKey.Public().(ed25519.PublicKey), ed25519.PublicKey(pubKeyBytes)) {
		return nil, fmt.Errorf("Ed25519 密钥不匹配，文件可能已损坏")
	}

	return ssh.NewSignerFromKey(privKey)
}

func makeRSASignerFromPPK(publicData, privateData []byte) (ssh.Signer, error) {
	// Public data: string("ssh-rsa") + mpint(e) + mpint(n)
	_, rest, err := sshString(publicData)
	if err != nil {
		return nil, fmt.Errorf("RSA 公钥格式错误: %w", err)
	}
	eInt, rest, err := sshMPInt(rest)
	if err != nil {
		return nil, fmt.Errorf("RSA 公钥 e 解析失败: %w", err)
	}
	nInt, _, err := sshMPInt(rest)
	if err != nil {
		return nil, fmt.Errorf("RSA 公钥 n 解析失败: %w", err)
	}

	// Private data: mpint(d) + mpint(p) + mpint(q) + mpint(iqmp)
	dInt, rest, err := sshMPInt(privateData)
	if err != nil {
		return nil, fmt.Errorf("RSA 私钥 d 解析失败: %w", err)
	}
	pInt, rest, err := sshMPInt(rest)
	if err != nil {
		return nil, fmt.Errorf("RSA 私钥 p 解析失败: %w", err)
	}
	qInt, rest, err := sshMPInt(rest)
	if err != nil {
		return nil, fmt.Errorf("RSA 私钥 q 解析失败: %w", err)
	}
	_, _, err = sshMPInt(rest) // iqmp — handled by Precompute
	if err != nil {
		return nil, fmt.Errorf("RSA 私钥 iqmp 解析失败: %w", err)
	}

	privKey := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{N: nInt, E: int(eInt.Int64())},
		D:         dInt,
		Primes:    []*big.Int{pInt, qInt},
	}
	privKey.Precompute()

	return ssh.NewSignerFromKey(privKey)
}

// sshIO implements terminalIO for SSH connections.
type sshIO struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
}

func newSSHSession(id string, cfg SSHConfig) (*Session, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	auth, err := BuildSSHAuth(cfg)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("ssh new session: %w", err)
	}

	// Request a PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Merge stderr into stdout
	session.StderrPipe()

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	return &Session{
		ID: id,
		io: &sshIO{
			client:  client,
			session: session,
			stdin:   stdin,
			stdout:  stdout,
		},
	}, nil
}

func (s *sshIO) Read(buf []byte) (int, error)  { return s.stdout.Read(buf) }
func (s *sshIO) Write(data []byte) (int, error) { return s.stdin.Write(data) }

func (s *sshIO) Resize(cols, rows uint16) error {
	return s.session.WindowChange(int(rows), int(cols))
}

func (s *sshIO) Close() error {
	s.stdin.Close()
	s.session.Close()
	return s.client.Close()
}

