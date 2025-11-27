# ATHENA OTA Client Library for Arduino

This library provides secure over-the-air (OTA) firmware update capabilities for Arduino devices (ESP32/ESP8266) connected to the ATHENA platform.

## Features

- **Secure Updates**: Cryptographic signature verification using RSA
- **Hash Verification**: SHA-256 hash checking to ensure firmware integrity
- **HTTPS Support**: Secure communication with OTA service
- **Progress Callbacks**: Real-time progress updates during download and installation
- **Status Reporting**: Automatic status reporting back to the OTA service
- **Error Handling**: Comprehensive error codes and messages
- **Flexible Configuration**: Support for different deployment strategies

## Requirements

### Hardware
- ESP32 or ESP8266 board
- WiFi connectivity
- Sufficient flash memory for OTA updates (typically 2MB+)

### Software Dependencies
- Arduino IDE 1.8.x or later (or PlatformIO)
- ESP32/ESP8266 Arduino Core
- ArduinoJson library (v6.x)
- Base64 library

### ATHENA Platform
- Device must be registered in ATHENA platform
- Device must be assigned to an OTA channel (stable, beta, or alpha)
- Valid device ID and authentication credentials

## Installation

### Arduino IDE

1. Download this library as a ZIP file
2. In Arduino IDE, go to Sketch → Include Library → Add .ZIP Library
3. Select the downloaded ZIP file
4. Install required dependencies via Library Manager:
   - ArduinoJson (by Benoit Blanchon)
   - Base64 (by Densaugeo)

### PlatformIO

Add to your `platformio.ini`:

```ini
lib_deps =
    bblanchon/ArduinoJson@^6.21.0
    densaugeo/base64@^1.4.0
```

## Quick Start

```cpp
#include <WiFi.h>
#include "OTAClient.h"

const char* ssid = "YOUR_WIFI_SSID";
const char* password = "YOUR_WIFI_PASSWORD";
const char* otaServerURL = "https://athena.example.com";
const char* deviceID = "YOUR_DEVICE_ID";
const char* publicKey = "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----";

OTAClient otaClient(otaServerURL, deviceID, publicKey);

void setup() {
    Serial.begin(115200);
    
    // Connect to WiFi
    WiFi.begin(ssid, password);
    while (WiFi.status() != WL_CONNECTED) {
        delay(500);
    }
    
    // Initialize OTA client
    otaClient.begin();
    
    // Check for updates
    if (otaClient.checkAndUpdate()) {
        Serial.println("Update successful! Rebooting...");
        ESP.restart();
    }
}

void loop() {
    // Your application code
}
```

## API Reference

### Constructor

```cpp
OTAClient(const char* serverURL, const char* deviceID, const char* publicKey)
```

Creates a new OTA client instance.

**Parameters:**
- `serverURL`: Base URL of the ATHENA OTA service
- `deviceID`: Unique device identifier
- `publicKey`: PEM-encoded RSA public key for signature verification

### Methods

#### `bool begin()`

Initializes the OTA client. Must be called before any other methods.

**Returns:** `true` if initialization successful, `false` otherwise

#### `bool checkForUpdate(FirmwareUpdate* update)`

Checks if a firmware update is available for this device.

**Parameters:**
- `update`: Pointer to FirmwareUpdate structure to fill with update information

**Returns:** `true` if update is available, `false` otherwise

#### `bool performUpdate(const FirmwareUpdate& update)`

Downloads and installs a firmware update.

**Parameters:**
- `update`: Firmware update information from `checkForUpdate()`

**Returns:** `true` if update successful, `false` otherwise

#### `bool checkAndUpdate()`

Convenience method that checks for update and installs if available.

**Returns:** `true` if update was performed successfully, `false` otherwise

#### `void setProgressCallback(ProgressCallback callback)`

Sets a callback function for download/installation progress updates.

**Parameters:**
- `callback`: Function with signature `void callback(size_t current, size_t total)`

#### `void setStatusCallback(StatusCallback callback)`

Sets a callback function for status changes.

**Parameters:**
- `callback`: Function with signature `void callback(const char* status, int progress)`

#### `void setCACertificate(const char* caCert)`

Sets the CA certificate for HTTPS verification.

**Parameters:**
- `caCert`: PEM-encoded CA certificate

#### `void setVerifySignature(bool enable)`

Enables or disables signature verification (enabled by default).

**Parameters:**
- `enable`: `true` to enable, `false` to disable (not recommended for production)

#### `int getLastError()`

Returns the last error code.

**Returns:** Error code (see Error Codes section)

#### `const char* getLastErrorMessage()`

Returns the last error message.

**Returns:** Error message string

## Error Codes

| Code | Constant | Description |
|------|----------|-------------|
| 0 | `OTA_ERROR_NONE` | No error |
| 1 | `OTA_ERROR_NO_UPDATE` | No update available |
| 2 | `OTA_ERROR_NETWORK` | Network communication error |
| 3 | `OTA_ERROR_DOWNLOAD` | Firmware download failed |
| 4 | `OTA_ERROR_VERIFICATION` | Hash or signature verification failed |
| 5 | `OTA_ERROR_INSTALLATION` | Firmware installation failed |
| 6 | `OTA_ERROR_INVALID_RESPONSE` | Invalid response from server |

## Update Status Flow

1. **Pending**: Update is available and queued
2. **Downloading**: Firmware binary is being downloaded
3. **Installing**: Firmware is being written to flash
4. **Completed**: Update completed successfully
5. **Failed**: Update failed (see error message for details)

## Security Considerations

### Signature Verification

The library uses RSA signature verification to ensure firmware authenticity. The public key must match the private key used by the ATHENA platform to sign firmware releases.

**Important:** Never disable signature verification in production environments.

### HTTPS/TLS

Always use HTTPS for communication with the OTA service. Set a valid CA certificate using `setCACertificate()` to prevent man-in-the-middle attacks.

### Memory Safety

The library allocates memory dynamically for firmware downloads. Ensure your device has sufficient free heap memory before performing updates.

## Examples

### Basic OTA

Simple example that checks for updates on startup and periodically.

See: `examples/BasicOTA/BasicOTA.ino`

### Advanced OTA

Advanced example with:
- Automatic retry on failure
- Health check after update
- Rollback detection
- Persistent update state

See: `examples/AdvancedOTA/AdvancedOTA.ino`

## Troubleshooting

### Update fails with "Hash verification failed"

- Ensure the firmware binary wasn't corrupted during download
- Check network connectivity
- Verify the binary hash in the ATHENA platform matches the actual firmware

### Update fails with "Signature verification failed"

- Verify the public key matches the private key used to sign the firmware
- Ensure the public key is in correct PEM format
- Check that the signature wasn't corrupted

### "Memory allocation failed" error

- Your device doesn't have enough free heap memory
- Try reducing memory usage in your application
- Consider using streaming updates (future feature)

### Device doesn't check for updates

- Verify WiFi connectivity
- Check that the device ID is correct
- Ensure the device is registered in the ATHENA platform
- Verify the OTA service URL is correct and accessible

### Update downloads but doesn't install

- Check that your device has sufficient flash space
- Verify the partition scheme supports OTA updates
- Ensure the firmware binary is compatible with your board

## Best Practices

1. **Always verify signatures** in production environments
2. **Use HTTPS** with valid CA certificates
3. **Implement health checks** after updates to detect issues
4. **Handle errors gracefully** and report failures to the server
5. **Test updates** on a small subset of devices before wide deployment
6. **Monitor update success rates** in the ATHENA dashboard
7. **Keep retry logic** to handle transient network failures
8. **Implement rollback detection** to prevent boot loops

## License

This library is part of the ATHENA platform and is licensed under the same terms as the main project.

## Support

For issues, questions, or contributions, please refer to the main ATHENA project repository.
