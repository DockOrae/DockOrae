package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// verifySHA256 校验下载文件的 SHA256 与 release 校验资产一致:
//   - 校验文件不存在(404,老版本 release 无 .sha256 资产)→ 跳过校验并告警
//   - 校验文件存在但内容不匹配 → 返回错误,中止更新(绝不替换当前运行文件)
func verifySHA256(filePath, sumURL string, client *http.Client) error {
	sumResp, err := client.Get(sumURL)
	if err != nil {
		return fmt.Errorf("下载校验文件失败: %w", err)
	}
	defer sumResp.Body.Close()
	if sumResp.StatusCode == http.StatusNotFound {
		log.Printf("update: 校验文件不存在(%s),跳过 SHA256 校验", sumURL)
		return nil
	}
	if sumResp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载校验文件失败: HTTP %d", sumResp.StatusCode)
	}
	sumData, err := io.ReadAll(io.LimitReader(sumResp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("读取校验文件失败: %w", err)
	}
	fields := strings.Fields(string(sumData))
	if len(fields) == 0 {
		return errors.New("校验文件格式无效")
	}
	expected := fields[0]

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%w: 期望 %s,实际 %s", ErrChecksumMismatch, expected, actual)
	}
	log.Printf("update: SHA256 校验通过")
	return nil
}
