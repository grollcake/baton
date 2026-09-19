package baton

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".baton-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		if runtime.GOOS != "windows" {
			return err
		}
		if _, statErr := os.Stat(path); statErr != nil {
			return err
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if renameErr := os.Rename(tempPath, path); renameErr != nil {
			return renameErr
		}
	}
	return nil
}

func copyFile(source, target string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := atomicWrite(target, data, mode); err != nil {
		return fmt.Errorf("copy %s to %s: %w", source, target, err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func checksumFor(checksumFile, relativePath string) (string, error) {
	file, err := os.Open(checksumFile)
	if err != nil {
		return "", err
	}
	defer file.Close()
	wanted := filepath.ToSlash(relativePath)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == wanted {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum not found for %s", wanted)
}

func verifyChecksum(path, checksumFile, relativePath string) error {
	expected, err := checksumFor(checksumFile, relativePath)
	if err != nil {
		return err
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s", relativePath)
	}
	return nil
}
