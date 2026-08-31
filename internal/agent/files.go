// Agent Client 宿主文件 + 终端端点(§55/§56)。
// 数据契约:Agent 返回 {ok,data} 信封,data 与前端 HostFile 结构对应。
package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
)

// HostFile 宿主文件条目(与 Agent files.Entry、前端 HostFile 对应)
type HostFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	ModifiedAt  string `json:"modified_at"`
	Mode        uint32 `json:"mode"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	Target      string `json:"target,omitempty"`
}

// FileListResult 目录列表结果
type FileListResult struct {
	Path    string     `json:"path"`
	Entries []HostFile `json:"entries"`
}

// FileCompressResult 压缩结果
type FileCompressResult struct {
	Archive    string `json:"archive"`
	Files      int    `json:"files"`
	Skipped    int    `json:"skipped"`
	SkippedWhy string `json:"skipped_why,omitempty"`
}

// FileExtractResult 解压结果
type FileExtractResult struct {
	Dest    string `json:"dest"`
	Files   int    `json:"files"`
	Skipped int    `json:"skipped"`
	Warning string `json:"warning,omitempty"`
}

// FileSearchResult 搜索结果
type FileSearchResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// callData 调用 Agent 并解码 data 字段到 out
func (c *Client) callData(ctx context.Context, method, path string, payload any, out any) error {
	data, err := c.Call(ctx, method, path, payload, "")
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// FilesList 目录列表(showHidden 控制隐藏文件)
func (c *Client) FilesList(ctx context.Context, path string, showHidden bool) (*FileListResult, error) {
	var res FileListResult
	if err := c.callData(ctx, http.MethodGet, "/v1/host/files/list?path="+url.QueryEscape(path)+"&show_hidden="+strconv.FormatBool(showHidden), nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FilesDirSize 目录大小
func (c *Client) FilesDirSize(ctx context.Context, path string) (int64, error) {
	var res struct {
		Size int64 `json:"size"`
	}
	if err := c.callData(ctx, http.MethodGet, "/v1/host/files/dirsize?path="+url.QueryEscape(path), nil, &res); err != nil {
		return 0, err
	}
	return res.Size, nil
}

// FilesChown 修改所有者/用户组
func (c *Client) FilesChown(ctx context.Context, path, owner, group string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/chown", map[string]any{"path": path, "owner": owner, "group": group}, nil)
}

// FilesStat 单条目属性
func (c *Client) FilesStat(ctx context.Context, path string) (*HostFile, error) {
	var e HostFile
	if err := c.callData(ctx, http.MethodGet, "/v1/host/files/stat?path="+url.QueryEscape(path), nil, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// FilesTouch 新建空文件
func (c *Client) FilesTouch(ctx context.Context, path string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/touch", map[string]any{"path": path}, nil)
}

// FilesMkdir 新建目录
func (c *Client) FilesMkdir(ctx context.Context, path string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/mkdir", map[string]any{"path": path}, nil)
}

// FilesRename 重命名
func (c *Client) FilesRename(ctx context.Context, oldPath, newPath string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/rename", map[string]any{"old_path": oldPath, "new_path": newPath}, nil)
}

// FilesCopy 复制
func (c *Client) FilesCopy(ctx context.Context, src, dst string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/copy", map[string]any{"src": src, "dst": dst}, nil)
}

// FilesMove 移动
func (c *Client) FilesMove(ctx context.Context, src, dst string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/move", map[string]any{"src": src, "dst": dst}, nil)
}

// FilesRemove 删除(confirm + recursive 由 Agent 强制校验;force=true 永久删除,否则回收站开启时进回收站)
func (c *Client) FilesRemove(ctx context.Context, paths []string, recursive, force bool) (bool, error) {
	var res struct {
		Trashed bool `json:"trashed"`
	}
	if err := c.callData(ctx, http.MethodPost, "/v1/host/files/remove", map[string]any{
		"paths": paths, "recursive": recursive, "force": force, "confirm": true,
	}, &res); err != nil {
		return false, err
	}
	return res.Trashed, nil
}

// TrashStatus 回收站状态
func (c *Client) TrashStatus(ctx context.Context) (map[string]any, error) {
	return c.Call(ctx, http.MethodGet, "/v1/host/files/trash/status", nil, "")
}

// TrashSetEnabled 回收站开关
func (c *Client) TrashSetEnabled(ctx context.Context, enabled bool) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/trash/enable", map[string]any{"enabled": enabled}, nil)
}

// TrashList 回收站列表
func (c *Client) TrashList(ctx context.Context) ([]TrashItem, error) {
	var res struct {
		Items []TrashItem `json:"items"`
	}
	if err := c.callData(ctx, http.MethodGet, "/v1/host/files/trash/list", nil, &res); err != nil {
		return nil, err
	}
	return res.Items, nil
}

// TrashRestore 恢复
func (c *Client) TrashRestore(ctx context.Context, names []string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/trash/restore", map[string]any{"names": names}, nil)
}

// TrashDelete 彻底删除
func (c *Client) TrashDelete(ctx context.Context, names []string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/trash/delete", map[string]any{"names": names, "confirm": true}, nil)
}

// TrashEmpty 清空
func (c *Client) TrashEmpty(ctx context.Context) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/trash/empty", map[string]any{"confirm": true}, nil)
}

// TrashItem 回收站条目(与 Agent trash.TrashItem 对应)
type TrashItem struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	Size       int64  `json:"size"`
	DeleteTime string `json:"delete_time"`
	IsDir      bool   `json:"is_dir"`
}

// FilesChmod 修改权限
func (c *Client) FilesChmod(ctx context.Context, path string, mode uint32) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/chmod", map[string]any{"path": path, "mode": mode}, nil)
}

// FilesWrite 覆盖写入(编辑器保存)
func (c *Client) FilesWrite(ctx context.Context, path, content string) error {
	return c.callData(ctx, http.MethodPost, "/v1/host/files/write", map[string]any{"path": path, "content": content}, nil)
}

// FilesCompress 压缩(tar.gz/zip)
func (c *Client) FilesCompress(ctx context.Context, dir, archive, format string, names []string) (*FileCompressResult, error) {
	var res FileCompressResult
	if err := c.callData(ctx, http.MethodPost, "/v1/host/files/compress", map[string]any{
		"dir": dir, "archive": archive, "format": format, "names": names,
	}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FilesExtract 解压
func (c *Client) FilesExtract(ctx context.Context, archive, dest string) (*FileExtractResult, error) {
	var res FileExtractResult
	if err := c.callData(ctx, http.MethodPost, "/v1/host/files/extract", map[string]any{"archive": archive, "dest": dest}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// FilesSearch 递归搜索
func (c *Client) FilesSearch(ctx context.Context, path, query string, limit int) ([]FileSearchResult, bool, error) {
	q := "/v1/host/files/search?path=" + url.QueryEscape(path) + "&q=" + url.QueryEscape(query)
	if limit > 0 {
		q += "&limit=" + strconv.Itoa(limit)
	}
	data, err := c.Call(ctx, http.MethodGet, q, nil, "")
	if err != nil {
		return nil, false, err
	}
	raw, _ := json.Marshal(data)
	var res struct {
		Results   []FileSearchResult `json:"results"`
		Truncated bool               `json:"truncated"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, false, err
	}
	return res.Results, res.Truncated, nil
}

// FilesDownloadRaw 下载(返回原始响应,调用方负责关闭 Body 并流式转发)
func (c *Client) FilesDownloadRaw(ctx context.Context, path string) (*http.Response, error) {
	return c.doRaw(ctx, http.MethodGet, "/v1/host/files/download?path="+url.QueryEscape(path), nil, "")
}

// FilesUploadRaw 上传(流式 body;Content-Type octet-stream)
func (c *Client) FilesUploadRaw(ctx context.Context, dir, name string, body io.Reader) (*http.Response, error) {
	return c.doRaw(ctx, http.MethodPost,
		"/v1/host/files/upload?dir="+url.QueryEscape(dir)+"&name="+url.QueryEscape(name), body, "application/octet-stream")
}

// doRaw 发起原始请求(不解析信封;上传/下载流式用)
func (c *Client) doRaw(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, body)
	if err != nil {
		return nil, &AgentError{Status: 500, Code: "INTERNAL", Message: "构造 Agent 请求失败"}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var env struct {
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Error != nil {
			return nil, &AgentError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
		}
		return nil, &AgentError{Status: resp.StatusCode, Code: "AGENT_ERROR", Message: string(raw)}
	}
	return resp, nil
}

// HostTerminalWS 宿主终端 WebSocket(cwd + 初始尺寸)
func (c *Client) HostTerminalWS(ctx context.Context, cwd string, cols, rows int) (*websocket.Conn, error) {
	q := ""
	if cwd != "" {
		q += "?cwd=" + url.QueryEscape(cwd)
	}
	if cols > 0 || rows > 0 {
		sep := "?"
		if q != "" {
			sep = "&"
		}
		q += sep + "cols=" + strconv.Itoa(cols) + "&rows=" + strconv.Itoa(rows)
	}
	return c.DialWS(ctx, "/v1/host/terminal/ws"+q)
}
