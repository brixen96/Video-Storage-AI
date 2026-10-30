package services

import (
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// registerJDownloaderTools exposes JDownloader CONTROL (not just status) to the
// AI Companion: queue inspection, start/stop/pause, speed limit, link sending,
// and linkgrabber management. The service already had all these methods — they
// simply had no tool surface.

func (s *AICompanionService) registerJDownloaderTools() {
	// ── jd_get_queue ──────────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "jd_get_queue",
			Description: "Get the live JDownloader queue: download packages with progress, linkgrabber packages waiting to be grabbed, current speed and state. Use when asked what JDownloader is doing or how downloads are going.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		jd := NewJDownloaderService()
		if !jd.IsAvailable() {
			return map[string]interface{}{"available": false, "note": "JDownloader is not reachable — check it is running and configured."}, nil
		}

		result := map[string]interface{}{"available": true}

		if state, err := jd.GetDownloadState(); err == nil {
			result["state"] = state
		}
		if speed, err := jd.GetDownloadSpeed(); err == nil {
			result["speed_bytes_per_sec"] = speed
		}
		if limit, err := jd.GetSpeedLimit(); err == nil {
			result["speed_limit_bytes_per_sec"] = limit
		}
		if pkgs, err := jd.GetDownloadPackages(); err == nil {
			result["download_packages"] = pkgs
		}
		if links, err := jd.GetDownloadLinks(); err == nil {
			result["download_links"] = links
		}
		if lgPkgs, err := jd.GetLinkGrabberPackages(); err == nil {
			result["linkgrabber_packages"] = lgPkgs
		}
		if lgLinks, err := jd.GetLinkGrabberLinks(); err == nil {
			result["linkgrabber_links"] = lgLinks
		}
		return result, nil
	})

	// ── jd_control ────────────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "jd_control",
			Description: "Control JDownloader: start, stop, or pause/resume downloads, set the speed limit, or clean up finished downloads. Use when Brix asks to start/stop/pause downloads or throttle bandwidth.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"start", "stop", "pause", "resume", "set_speed_limit", "cleanup_finished"},
					"description": "Control action to perform",
				},
				"speed_limit_bps": map[string]interface{}{
					"type":        "number",
					"description": "Bytes per second — required for set_speed_limit (0 = unlimited)",
				},
			}, "required": []string{"action"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		action, _ := args["action"].(string)
		jd := NewJDownloaderService()
		if !jd.IsAvailable() {
			return map[string]interface{}{"error": "JDownloader is not reachable"}, nil
		}

		switch action {
		case "start":
			if err := jd.StartDownloads(); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "start", "ok": true}, nil
		case "stop":
			if err := jd.StopDownloads(); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "stop", "ok": true}, nil
		case "pause":
			if err := jd.PauseDownloads(true); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "pause", "ok": true}, nil
		case "resume":
			if err := jd.PauseDownloads(false); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "resume", "ok": true}, nil
		case "set_speed_limit":
			limit := int64(floatArg(args, "speed_limit_bps", 0))
			if err := jd.SetSpeedLimit(limit); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "set_speed_limit", "limit_bps": limit, "ok": true}, nil
		case "cleanup_finished":
			if err := jd.CleanupFinished(); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": "cleanup_finished", "ok": true}, nil
		default:
			return map[string]interface{}{"error": "unknown action: " + action}, nil
		}
	})

	// ── jd_send_links ─────────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "jd_send_links",
			Description: "Send URLs to JDownloader's linkgrabber (or directly to downloads with start=true). Use when Brix gives you links to download or asks you to add links to JDownloader.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"links": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of URLs to send",
				},
				"package_name": map[string]interface{}{
					"type":        "string",
					"description": "Optional package name to group the links under",
				},
				"start": map[string]interface{}{
					"type":        "boolean",
					"description": "Start downloading immediately instead of just grabbing (default: false)",
				},
			}, "required": []string{"links"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawLinks, _ := args["links"].([]interface{})
		if len(rawLinks) == 0 {
			return map[string]interface{}{"error": "links array required"}, nil
		}
		pkgName, _ := args["package_name"].(string)
		startNow, _ := args["start"].(bool)

		jd := NewJDownloaderService()
		if !jd.IsAvailable() {
			return map[string]interface{}{"error": "JDownloader is not reachable"}, nil
		}

		linkReqs := make([]models.JDownloaderLinkRequest, 0, len(rawLinks))
		for _, l := range rawLinks {
			url, _ := l.(string)
			if strings.TrimSpace(url) == "" {
				continue
			}
			linkReqs = append(linkReqs, models.JDownloaderLinkRequest{
				URL:         url,
				PackageName: pkgName,
				AutoStart:   startNow,
			})
		}
		if len(linkReqs) == 0 {
			return map[string]interface{}{"error": "no valid URLs in links array"}, nil
		}

		results, err := jd.SendLinks(linkReqs)
		if err != nil {
			return nil, err
		}

		sent := 0
		failed := 0
		for _, r := range results {
			if r.Success {
				sent++
			} else {
				failed++
			}
		}

		return map[string]interface{}{
			"sent":    sent,
			"failed":  failed,
			"started": startNow,
			"results": results,
		}, nil
	})

	// ── jd_linkgrabber_action ─────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "jd_linkgrabber_action",
			Description: "Manage JDownloader's linkgrabber: move grabbed links to downloads, remove links, or clear the linkgrabber entirely. Use after links are grabbed and ready to start, or to clean up stale links.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"move_to_downloads", "clear_all", "remove_links"},
					"description": "Linkgrabber action",
				},
				"link_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "Link IDs — required for remove_links",
				},
				"package_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "Package IDs — optional, for package-level actions",
				},
			}, "required": []string{"action"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		action, _ := args["action"].(string)
		jd := NewJDownloaderService()
		if !jd.IsAvailable() {
			return map[string]interface{}{"error": "JDownloader is not reachable"}, nil
		}

		linkIDs := toInt64Slice(args["link_ids"])
		pkgIDs := toInt64Slice(args["package_ids"])

		switch action {
		case "move_to_downloads":
			if err := jd.MoveToDownloads(linkIDs, pkgIDs); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "ok": true, "links": len(linkIDs), "packages": len(pkgIDs)}, nil
		case "clear_all":
			if err := jd.ClearLinkGrabber(); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "ok": true}, nil
		case "remove_links":
			if len(linkIDs) == 0 && len(pkgIDs) == 0 {
				return map[string]interface{}{"error": "link_ids or package_ids required for remove_links"}, nil
			}
			if err := jd.RemoveLinkGrabberLinks(linkIDs, pkgIDs); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "ok": true, "removed_links": len(linkIDs), "removed_packages": len(pkgIDs)}, nil
		default:
			return map[string]interface{}{"error": "unknown action: " + action}, nil
		}
	})
}

// toInt64Slice converts a JSON array of numbers to []int64.
func toInt64Slice(v interface{}) []int64 {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(arr))
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			out = append(out, int64(n))
		case int64:
			out = append(out, n)
		case int:
			out = append(out, int64(n))
		}
	}
	return out
}