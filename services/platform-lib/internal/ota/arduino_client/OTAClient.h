#ifndef OTA_CLIENT_H
#define OTA_CLIENT_H

#include <Arduino.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <Update.h>
#include <mbedtls/sha256.h>
#include <mbedtls/pk.h>

// Update status constants
#define OTA_STATUS_PENDING "pending"
#define OTA_STATUS_DOWNLOADING "downloading"
#define OTA_STATUS_INSTALLING "installing"
#define OTA_STATUS_COMPLETED "completed"
#define OTA_STATUS_FAILED "failed"

// Error codes
#define OTA_ERROR_NONE 0
#define OTA_ERROR_NO_UPDATE 1
#define OTA_ERROR_NETWORK 2
#define OTA_ERROR_DOWNLOAD 3
#define OTA_ERROR_VERIFICATION 4
#define OTA_ERROR_INSTALLATION 5
#define OTA_ERROR_INVALID_RESPONSE 6

/**
 * @brief Structure to hold firmware update information
 */
struct FirmwareUpdate {
    String releaseID;
    String version;
    String binaryURL;
    String binaryHash;
    int64_t binarySize;
    String signature;
    String releaseNotes;
};

/**
 * @brief Callback function type for update progress
 * @param current Current bytes downloaded/installed
 * @param total Total bytes to download/install
 */
typedef void (*ProgressCallback)(size_t current, size_t total);

/**
 * @brief Callback function type for status changes
 * @param status Current status string
 * @param progress Progress percentage (0-100)
 */
typedef void (*StatusCallback)(const char* status, int progress);

/**
 * @brief OTA Client for Arduino devices
 * 
 * This class handles secure over-the-air firmware updates for Arduino devices.
 * It communicates with the ATHENA OTA service to check for updates, download
 * firmware, verify signatures, and install updates.
 */
class OTAClient {
public:
    /**
     * @brief Construct a new OTAClient object
     * 
     * @param serverURL Base URL of the OTA service (e.g., "https://athena.example.com")
     * @param deviceID Unique device identifier
     * @param publicKey PEM-encoded public key for signature verification
     */
    OTAClient(const char* serverURL, const char* deviceID, const char* publicKey);
    
    /**
     * @brief Destroy the OTAClient object
     */
    ~OTAClient();
    
    /**
     * @brief Initialize the OTA client
     * 
     * @return true if initialization successful
     * @return false if initialization failed
     */
    bool begin();
    
    /**
     * @brief Check if an update is available for this device
     * 
     * @param update Pointer to FirmwareUpdate structure to fill with update info
     * @return true if update is available
     * @return false if no update available or error occurred
     */
    bool checkForUpdate(FirmwareUpdate* update);
    
    /**
     * @brief Download and install a firmware update
     * 
     * @param update Firmware update information
     * @return true if update successful
     * @return false if update failed
     */
    bool performUpdate(const FirmwareUpdate& update);
    
    /**
     * @brief Check for update and install if available (convenience method)
     * 
     * @return true if update was performed successfully
     * @return false if no update or update failed
     */
    bool checkAndUpdate();
    
    /**
     * @brief Set progress callback for download/installation progress
     * 
     * @param callback Function to call with progress updates
     */
    void setProgressCallback(ProgressCallback callback);
    
    /**
     * @brief Set status callback for status changes
     * 
     * @param callback Function to call with status updates
     */
    void setStatusCallback(StatusCallback callback);
    
    /**
     * @brief Get the last error code
     * 
     * @return int Error code (OTA_ERROR_*)
     */
    int getLastError() const;
    
    /**
     * @brief Get the last error message
     * 
     * @return const char* Error message string
     */
    const char* getLastErrorMessage() const;
    
    /**
     * @brief Set CA certificate for HTTPS verification
     * 
     * @param caCert PEM-encoded CA certificate
     */
    void setCACertificate(const char* caCert);
    
    /**
     * @brief Enable or disable signature verification
     * 
     * @param enable true to enable, false to disable (not recommended)
     */
    void setVerifySignature(bool enable);

private:
    String _serverURL;
    String _deviceID;
    String _publicKey;
    String _caCert;
    bool _verifySignature;
    
    int _lastError;
    String _lastErrorMessage;
    
    ProgressCallback _progressCallback;
    StatusCallback _statusCallback;
    
    WiFiClientSecure _wifiClient;
    HTTPClient _httpClient;
    
    /**
     * @brief Report update status to the server
     * 
     * @param releaseID Release ID
     * @param status Status string
     * @param progress Progress percentage
     * @param errorMessage Error message (if any)
     * @return true if report successful
     * @return false if report failed
     */
    bool reportStatus(const String& releaseID, const char* status, int progress, const char* errorMessage = nullptr);
    
    /**
     * @brief Download firmware binary from URL
     * 
     * @param url Download URL
     * @param buffer Buffer to store downloaded data
     * @param expectedSize Expected size of download
     * @return size_t Actual bytes downloaded (0 on error)
     */
    size_t downloadFirmware(const String& url, uint8_t** buffer, size_t expectedSize);
    
    /**
     * @brief Verify firmware hash
     * 
     * @param data Firmware data
     * @param size Data size
     * @param expectedHash Expected SHA-256 hash (hex string)
     * @return true if hash matches
     * @return false if hash doesn't match
     */
    bool verifyHash(const uint8_t* data, size_t size, const String& expectedHash);
    
    /**
     * @brief Verify firmware signature
     * 
     * @param data Firmware data
     * @param size Data size
     * @param signature Base64-encoded signature
     * @return true if signature is valid
     * @return false if signature is invalid
     */
    bool verifySignature(const uint8_t* data, size_t size, const String& signature);
    
    /**
     * @brief Install firmware update
     * 
     * @param data Firmware data
     * @param size Data size
     * @return true if installation successful
     * @return false if installation failed
     */
    bool installFirmware(const uint8_t* data, size_t size);
    
    /**
     * @brief Set error state
     * 
     * @param errorCode Error code
     * @param errorMessage Error message
     */
    void setError(int errorCode, const String& errorMessage);
    
    /**
     * @brief Compute SHA-256 hash of data
     * 
     * @param data Input data
     * @param size Data size
     * @param output Output buffer (must be 32 bytes)
     */
    void computeSHA256(const uint8_t* data, size_t size, uint8_t* output);
    
    /**
     * @brief Convert hex string to bytes
     * 
     * @param hex Hex string
     * @param bytes Output byte array
     * @param maxLen Maximum length of output
     * @return size_t Number of bytes converted
     */
    size_t hexToBytes(const String& hex, uint8_t* bytes, size_t maxLen);
    
    /**
     * @brief Decode base64 string
     * 
     * @param input Base64 string
     * @param output Output buffer
     * @param maxLen Maximum output length
     * @return size_t Number of bytes decoded
     */
    size_t base64Decode(const String& input, uint8_t* output, size_t maxLen);
};

#endif // OTA_CLIENT_H
