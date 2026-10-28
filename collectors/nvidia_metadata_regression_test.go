package collectors

import (
	"encoding/json"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"testing"
	"time"
)

func TestNvidiaOptionalMetadataFailureDoesNotAbortInitialization(t *testing.T) {
	for _, option := range []string{"add_board_number_meta", "add_serial_meta", "add_uuid_meta"} {
		t.Run(option, func(t *testing.T) {
			_, stops := mockNvidiaLifecycle(t)
			handle, pci, board, serial, uuid := nvml.DeviceGetHandleByIndex, nvml.DeviceGetPciInfo, nvml.DeviceGetBoardPartNumber, nvml.DeviceGetSerial, nvml.DeviceGetUUID
			t.Cleanup(func() {
				nvml.DeviceGetHandleByIndex = handle
				nvml.DeviceGetPciInfo = pci
				nvml.DeviceGetBoardPartNumber = board
				nvml.DeviceGetSerial = serial
				nvml.DeviceGetUUID = uuid
			})
			nvml.DeviceGetCount = func() (int, nvml.Return) { return 1, nvml.SUCCESS }
			nvml.DeviceGetHandleByIndex = func(int) (nvml.Device, nvml.Return) { return nil, nvml.SUCCESS }
			nvml.DeviceGetPciInfo = func(nvml.Device) (nvml.PciInfo, nvml.Return) { return nvml.PciInfo{}, nvml.SUCCESS }
			nvml.DeviceGetBoardPartNumber = func(nvml.Device) (string, nvml.Return) { return "", nvml.ERROR_NOT_SUPPORTED }
			nvml.DeviceGetSerial = func(nvml.Device) (string, nvml.Return) { return "", nvml.ERROR_NOT_SUPPORTED }
			nvml.DeviceGetUUID = func(nvml.Device) (string, nvml.Return) { return "", nvml.ERROR_NOT_SUPPORTED }
			collector := NvidiaCollector{}
			err := collector.Init(json.RawMessage(`{"` + option + `":true}`))
			if err != nil || !collector.Initialized() || collector.num_gpus != 1 {
				t.Fatalf("optional failure aborted initialization err=%v init=%v count=%d", err, collector.Initialized(), collector.num_gpus)
			}
			collector.Close()
			if *stops != 1 {
				t.Fatalf("successful optional failure leaked ownership: %d", *stops)
			}
		})
	}
}
func TestNvidiaMigUuidFailureUsesIndexFallback(t *testing.T) {
	name, isMig, link, migMode, maxMig, migHandle, uuid, temp, errorString := nvml.DeviceGetName, nvml.DeviceIsMigDeviceHandle, nvml.DeviceGetNvLinkState, nvml.DeviceGetMigMode, nvml.DeviceGetMaxMigDeviceCount, nvml.DeviceGetMigDeviceHandleByIndex, nvml.DeviceGetUUID, nvml.DeviceGetTemperature, nvml.ErrorString
	t.Cleanup(func() {
		nvml.DeviceGetName = name
		nvml.DeviceIsMigDeviceHandle = isMig
		nvml.DeviceGetNvLinkState = link
		nvml.DeviceGetMigMode = migMode
		nvml.DeviceGetMaxMigDeviceCount = maxMig
		nvml.DeviceGetMigDeviceHandleByIndex = migHandle
		nvml.DeviceGetUUID = uuid
		nvml.DeviceGetTemperature = temp
		nvml.ErrorString = errorString
	})
	nvml.DeviceGetName = func(nvml.Device) (string, nvml.Return) { return "mock GPU", nvml.SUCCESS }
	nvml.DeviceIsMigDeviceHandle = func(nvml.Device) (bool, nvml.Return) { return false, nvml.SUCCESS }
	nvml.DeviceGetNvLinkState = func(nvml.Device, int) (nvml.EnableState, nvml.Return) { return nvml.FEATURE_DISABLED, nvml.SUCCESS }
	nvml.DeviceGetMigMode = func(nvml.Device) (int, int, nvml.Return) { return nvml.DEVICE_MIG_ENABLE, 0, nvml.SUCCESS }
	nvml.DeviceGetMaxMigDeviceCount = func(nvml.Device) (int, nvml.Return) { return 1, nvml.SUCCESS }
	nvml.DeviceGetMigDeviceHandleByIndex = func(nvml.Device, int) (nvml.Device, nvml.Return) { return nil, nvml.SUCCESS }
	nvml.DeviceGetUUID = func(nvml.Device) (string, nvml.Return) { return "", nvml.ERROR_NOT_SUPPORTED }
	nvml.DeviceGetTemperature = func(nvml.Device, nvml.TemperatureSensors) (uint32, nvml.Return) { return 35, nvml.SUCCESS }
	nvml.ErrorString = func(nvml.Return) string { return "mock UUID unsupported" }
	names := []string{"nv_average_power", "nv_bar1_mem_total", "nv_bar1_mem_used", "nv_compute_processes", "nv_decoder_util", "nv_ecc_corrected_error", "nv_ecc_mode", "nv_ecc_uncorrected_error", "nv_encoder_util", "nv_energy", "nv_energy_abs", "nv_fan", "nv_fb_mem_reserved", "nv_fb_mem_total", "nv_fb_mem_used", "nv_graphics_clock", "nv_graphics_processes", "nv_max_graphics_clock", "nv_max_mem_clock", "nv_max_sm_clock", "nv_max_video_clock", "nv_mem_clock", "nv_mem_util", "nv_mps_compute_processes", "nv_nvlink_crc_errors", "nv_nvlink_crc_errors_sum", "nv_nvlink_crc_flit_errors", "nv_nvlink_crc_flit_errors_sum", "nv_nvlink_ecc_errors", "nv_nvlink_ecc_errors_sum", "nv_nvlink_recovery_errors", "nv_nvlink_recovery_errors_sum", "nv_nvlink_replay_errors", "nv_nvlink_replay_errors_sum", "nv_perf_state", "nv_power_max_limit", "nv_power_usage", "nv_remapped_rows_corrected", "nv_remapped_rows_failure", "nv_remapped_rows_pending", "nv_remapped_rows_uncorrected", "nv_sm_clock", "nv_util", "nv_video_clock", "nv_violation_below_app_clock", "nv_violation_below_base_clock", "nv_violation_board_limit", "nv_violation_low_util", "nv_violation_power", "nv_violation_reliability", "nv_violation_sync_boost", "nv_violation_thermal"}
	excluded := map[string]bool{}
	for _, n := range names {
		excluded[n] = true
	}
	collector := NvidiaCollector{metricCollector: metricCollector{init: true, name: "fixture Nvidia"}, num_gpus: 1, config: NvidiaCollectorConfig{ProcessMigDevices: true, UseUuidForMigDevices: true, ExcludeMetrics: names}, gpus: []NvidiaCollectorDevice{{excludeMetrics: excluded, tags: map[string]string{"type": "accelerator", "type-id": "0"}, meta: map[string]string{}}}}
	out := make(chan lp.CCMessage, 8)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MIG UUID failure panicked: %v", r)
		}
	}()
	collector.Read(time.Second, out)
	if len(out) != 2 {
		t.Fatalf("expected physical+MIG temperature got %d", len(out))
	}
	physical, mig := <-out, <-out
	if physical.Name() != "nv_temp" || mig.Name() != "nv_temp" {
		t.Fatalf("unexpected names %s %s", physical.Name(), mig.Name())
	}
	if v, _ := mig.GetTag("stype-id"); v != "0" {
		t.Errorf("UUID fallback stype-id %q", v)
	}
	if v, _ := mig.GetTag("stype"); v != "mig" {
		t.Errorf("UUID fallback stype %q", v)
	}
}
