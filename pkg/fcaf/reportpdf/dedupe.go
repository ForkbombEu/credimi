// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import "regexp"

// maestroBurstScreenshot matches the per-command screenshots a mobile runner
// stores for one Maestro flow. Each command in a flow stores one such shot
// while the screen barely changes, so the report keeps only the final shot of
// each burst. Two runner naming schemes exist:
//   - local runner: obtain_pid_sdjwt_screenshot_1788207713069_action_rzl856vepb.yaml2295819707.png
//   - cloud runner: obtain_pid_sdjwt_step_004_tap_on_element_eudi_wallet_c0pcg1tjd1.png
var maestroBurstScreenshot = regexp.MustCompile(
	`^(.+)(?:_screenshot_[0-9]+_action_[A-Za-z0-9_]+\.yaml[0-9]+|_step_[0-9]+_[A-Za-z0-9_]+)\.png$`,
)

// DeduplicateScreenshots keeps one screenshot per Maestro command burst (the
// last, which captures the final state of the step) and every non-burst
// screenshot unchanged. It returns the kept images and the dropped count.
func DeduplicateScreenshots(images []ImageAsset) ([]ImageAsset, int) {
	if len(images) < 2 {
		return images, 0
	}
	lastOfBurst := map[string]int{}
	burstPrefix := make([]string, len(images))
	for index, image := range images {
		match := maestroBurstScreenshot.FindStringSubmatch(image.Filename)
		if match == nil {
			continue
		}
		burstPrefix[index] = match[1]
		lastOfBurst[match[1]] = index
	}

	kept := make([]ImageAsset, 0, len(images))
	dropped := 0
	for index, image := range images {
		if prefix := burstPrefix[index]; prefix != "" && lastOfBurst[prefix] != index {
			dropped++
			continue
		}
		kept = append(kept, image)
	}
	return kept, dropped
}
