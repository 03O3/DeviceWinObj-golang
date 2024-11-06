package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	DIGCF_PRESENT                     = 0x00000002
	DIGCF_ALLCLASSES                  = 0x00000004
	SPDRP_DEVICEDESC                  = 0x00000000
	SPDRP_HARDWAREID                  = 0x00000001
	SPDRP_MFG                         = 0x0000000B
	SPDRP_DRIVER                      = 0x00000009
	SPDRP_PHYSICAL_DEVICE_OBJECT_NAME = 0x0000000E // Свойство для физического имени
	INVALID_HANDLE_VALUE              = ^uintptr(0)
)

var (
	setupapi                             = syscall.NewLazyDLL("setupapi.dll")
	procSetupDiGetClassDevs              = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInfo            = setupapi.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiGetDeviceRegistryProperty = setupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	procSetupDiDestroyDeviceInfoList     = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

type SP_DEVINFO_DATA struct {
	CbSize    uint32
	ClassGuid windows.GUID
	DevInst   uint32
	Reserved  uintptr
}

func getDeviceProperty(deviceInfoSet uintptr, deviceInfoData *SP_DEVINFO_DATA, property uint32) (string, error) {
	var propertyBuffer [256]uint16
	var requiredSize uint32

	ret, _, _ := procSetupDiGetDeviceRegistryProperty.Call(
		deviceInfoSet,
		uintptr(unsafe.Pointer(deviceInfoData)),
		uintptr(property),
		0,
		uintptr(unsafe.Pointer(&propertyBuffer[0])),
		uintptr(len(propertyBuffer)*2),
		uintptr(unsafe.Pointer(&requiredSize)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to get device property %d", property)
	}

	return syscall.UTF16ToString(propertyBuffer[:]), nil
}

func findAndDisplayDeviceInfo(deviceName string) (bool, error) {
	deviceInfoSet, _, _ := procSetupDiGetClassDevs.Call(
		0,
		0,
		0,
		DIGCF_PRESENT|DIGCF_ALLCLASSES,
	)
	if deviceInfoSet == INVALID_HANDLE_VALUE {
		return false, fmt.Errorf("failed to get device info set")
	}
	defer procSetupDiDestroyDeviceInfoList.Call(deviceInfoSet)

	var deviceIndex uint32

	for {
		var deviceInfoData SP_DEVINFO_DATA
		deviceInfoData.CbSize = uint32(unsafe.Sizeof(deviceInfoData))

		ret, _, _ := procSetupDiEnumDeviceInfo.Call(
			deviceInfoSet,
			uintptr(deviceIndex),
			uintptr(unsafe.Pointer(&deviceInfoData)),
		)

		if ret == 0 {
			break
		}

		desc, err := getDeviceProperty(deviceInfoSet, &deviceInfoData, SPDRP_DEVICEDESC)
		if err != nil || desc != deviceName {
			deviceIndex++
			continue
		}

		fmt.Printf("Устройство \"%s\" найдено:\n", deviceName)

		if desc != "" {
			fmt.Printf("  Описание: %s\n", desc)
		}

		hardwareID, err := getDeviceProperty(deviceInfoSet, &deviceInfoData, SPDRP_HARDWAREID)
		if err == nil {
			fmt.Printf("  ID оборудования: %s\n", hardwareID)
		}

		manufacturer, err := getDeviceProperty(deviceInfoSet, &deviceInfoData, SPDRP_MFG)
		if err == nil {
			fmt.Printf("  Производитель: %s\n", manufacturer)
		}

		driver, err := getDeviceProperty(deviceInfoSet, &deviceInfoData, SPDRP_DRIVER)
		if err == nil {
			fmt.Printf("  Драйвер: %s\n", driver)
		}

		physicalName, err := getDeviceProperty(deviceInfoSet, &deviceInfoData, SPDRP_PHYSICAL_DEVICE_OBJECT_NAME)
		if err == nil {
			fmt.Printf("  Физическое имя: %s\n", physicalName)
		}

		return true, nil
	}

	fmt.Printf("Устройство \"%s\" не найдено.\n", deviceName)
	return false, nil
}

func main() {
	deviceName := "Logitech G HUB Virtual Bus Enumerator"
	_, err := findAndDisplayDeviceInfo(deviceName)
	fmt.Println("Press the Enter Key to terminate the console screen!")
	fmt.Scanln()
	if err != nil {
		fmt.Println("Ошибка:", err)
	}
}
