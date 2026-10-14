package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rancher-logs/config"
	"rancher-logs/logger"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	clusterID  string
}

type ClusterInfo struct {
	ID          string
	DisplayName string
}

func NewClient(cfg *config.Config) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		baseURL:  strings.TrimRight(cfg.RancherURL, "/"),
		username: cfg.Username,
		password: cfg.Password,
	}
}

func (c *Client) SetCluster(id string) {
	c.clusterID = id
}

func (c *Client) GetClusters() ([]ClusterInfo, error) {
	url := fmt.Sprintf("%s/v3/clusters", c.baseURL)
	data, err := c.doGet(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		logger.Errorf("JSON parse error (clusters): %v", err)
		return nil, fmt.Errorf("failed to parse clusters response: %w", err)
	}

	clusters := make([]ClusterInfo, 0, len(resp.Data))
	for _, c := range resp.Data {
		name := c.Name
		if name == "" {
			name = c.ID
		}
		clusters = append(clusters, ClusterInfo{ID: c.ID, DisplayName: name})
	}
	return clusters, nil
}

func isRetryable(statusCode int) bool {
	return statusCode == 502 || statusCode == 503 || statusCode == 504
}

func (c *Client) doGet(fullURL string) ([]byte, error) {
	logger.Info(fmt.Sprintf("GET %s", fullURL))

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * 2 * time.Second
			logger.Info(fmt.Sprintf("Retry %d/%d after %v...", attempt, maxRetries, delay))
			time.Sleep(delay)
		}

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.SetBasicAuth(c.username, c.password)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("connection error: %w", err)
			logger.Errorf("request failed (attempt %d): %v", attempt+1, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			logger.Errorf("auth error: HTTP %d", resp.StatusCode)
			return nil, fmt.Errorf("authentication failed (HTTP %d) - check username/password", resp.StatusCode)
		}

		if isRetryable(resp.StatusCode) {
			lastErr = fmt.Errorf("server returned HTTP %d", resp.StatusCode)
			logger.Errorf("HTTP %d (attempt %d): %s", resp.StatusCode, attempt+1, string(body))
			continue
		}

		if resp.StatusCode != 200 {
			logger.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
		}

		return body, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// Rancher v3 API response for namespaces
type NamespaceListV3 struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

func (c *Client) GetNamespaces() ([]string, error) {
	url := fmt.Sprintf("%s/v3/clusters/%s/namespaces", c.baseURL, c.clusterID)
	data, err := c.doGet(url)
	if err != nil {
		return nil, err
	}

	var nsList NamespaceListV3
	if err := json.Unmarshal(data, &nsList); err != nil {
		logger.Errorf("JSON parse error (namespaces): %v", err)
		return nil, fmt.Errorf("failed to parse namespaces response: %w", err)
	}

	names := make([]string, 0, len(nsList.Data))
	for _, item := range nsList.Data {
		if item.Name != "" {
			names = append(names, item.Name)
		} else if item.ID != "" {
			names = append(names, item.ID)
		}
	}
	return names, nil
}

type PodList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type PodInfo struct {
	Name   string
	Status string
}

func (c *Client) GetPods(namespace string) ([]PodInfo, error) {
	url := fmt.Sprintf("%s/k8s/clusters/%s/api/v1/namespaces/%s/pods", c.baseURL, c.clusterID, namespace)
	data, err := c.doGet(url)
	if err != nil {
		return nil, err
	}

	var podList PodList
	if err := json.Unmarshal(data, &podList); err != nil {
		logger.Errorf("JSON parse error (pods): %v", err)
		return nil, fmt.Errorf("failed to parse pods response: %w", err)
	}

	pods := make([]PodInfo, 0, len(podList.Items))
	for _, item := range podList.Items {
		pods = append(pods, PodInfo{
			Name:   item.Metadata.Name,
			Status: item.Status.Phase,
		})
	}
	return pods, nil
}

func (c *Client) StreamLogs(namespace, podName string, tailLines int) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/k8s/clusters/%s/api/v1/namespaces/%s/pods/%s/log?follow=true&timestamps=true&tailLines=%d",
		c.baseURL, c.clusterID, namespace, podName, tailLines)

	logger.Info(fmt.Sprintf("STREAM %s", url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	// No timeout for streaming
	streamClient := &http.Client{
		Transport: c.httpClient.Transport,
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		logger.Errorf("stream connection failed: %v", err)
		return nil, fmt.Errorf("stream connection error: %w", err)
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		resp.Body.Close()
		return nil, fmt.Errorf("authentication failed (HTTP %d) - check username/password", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logger.Errorf("stream HTTP %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *Client) FetchAllLogs(namespace, podName string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/k8s/clusters/%s/api/v1/namespaces/%s/pods/%s/log?timestamps=true",
		c.baseURL, c.clusterID, namespace, podName)

	logger.Info(fmt.Sprintf("FETCH ALL LOGS %s", url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	// Use a longer timeout for full log fetch
	fetchClient := &http.Client{
		Transport: c.httpClient.Transport,
		Timeout:   120 * time.Second,
	}

	resp, err := fetchClient.Do(req)
	if err != nil {
		logger.Errorf("fetch all logs failed: %v", err)
		return nil, fmt.Errorf("fetch all logs error: %w", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		logger.Errorf("fetch all logs HTTP %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}
