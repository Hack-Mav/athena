#include "OTAClient.h"
#include <ArduinoJson.h>
#include <base64.h>

OTAClient::OTAClient(const char* serverURL, const char* deviceID, const char* publicKey)
    : _serverURL(serverURL), _deviceID(deviceID), _publicKey(publicKey),
      _verifySignature(true), _lastError(OTA_ERROR_NONE),
      _progressCallback(nullptr), _statusCallback(nullptr) {
}

OTAClient::~OTAClient() {
    _httpClient.end();
}

bool OTAClient::begin() {
    // Configure WiFi client for HTTPS
    if (_caCert.length() > 0) {
        _wifiClient.setCACert(_caCert.c_str());
    } else {
        // If no CA cert provided, skip verification (not recommended for production)
        _wifiClient.setInsecure();
    }
    
    return true;
}

bool OTAClient::checkForUpdate(FirmwareUpdate* update) {
    if (!update) {
        setError(OTA_ERROR_INVALID_RESPONSE, "Invalid update pointer");
        return false;
    }
    
    // Build URL for checking updates
    String url = _serverURL + "/api/v1/ota/updates/" + _deviceID;
    
    _httpClient.begin(_wifiClient, url);
    _httpClient.addHeader("Content-Type", "application/json");
    
    int httpCode = _httpClient.GET();
    
    if (httpCode != HTTP_CODE_OK) {
        _httpClient.end();
        if (httpCode == HTTP_CODE_NOT_FOUND) {
            setError(OTA_ERROR_NO_UPDATE, "No update available");
        } else {
            setError(OTA_ERROR_NETWORK, "HTTP error: " + String(httpCode));
        }
        return false;
    }
    
    String payload = _httpClient.getString();
    _httpClient.end();
    
    // Parse JSON response
    DynamicJsonDocument doc(2048);
    DeserializationError error = deserializeJson(doc, payload);
    
    if (error) {
        setError(OTA_ERROR_INVALID_RESPONSE, "JSON parse error: " + String(error.c_str()));
        return false;
    }
    
    // Extract update information
    update->releaseID = doc["release_id"].as<String>();
    update->version = doc["version"].as<String>();
    update->binaryURL = doc["binary_url"].as<String>();
    update->binaryHash = doc["binary_hash"].as<String>();
    update->binarySize = doc["binary_size"].as<int64_t>();
    update->signature = doc["signature"].as<String>();
    update->releaseNotes = doc["release_notes"].as<String>();
    
    // Validate required fields
    if (update->releaseID.length() == 0 || update->binaryURL.length() == 0 || 
        update->binaryHash.length() == 0) {
        setError(OTA_ERROR_INVALID_RESPONSE, "Missing required fields in update response");
        return false;
    }
    
    _lastError = OTA_ERROR_NONE;
    return true;
}

bool OTAClient::performUpdate(const FirmwareUpdate& update) {
    uint8_t* firmwareData = nullptr;
    bool success = false;
    
    // Report downloading status
    reportStatus(update.releaseID, OTA_STATUS_DOWNLOADING, 0);
    if (_statusCallback) {
        _statusCallback(OTA_STATUS_DOWNLOADING, 0);
    }
    
    // Download firmware
    size_t downloadedSize = downloadFirmware(update.binaryURL, &firmwareData, update.binarySize);
    if (downloadedSize == 0 || firmwareData == nullptr) {
        reportStatus(update.releaseID, OTA_STATUS_FAILED, 0, _lastErrorMessage.c_str());
        goto cleanup;
    }
    
    // Verify hash
    if (!verifyHash(firmwareData, downloadedSize, update.binaryHash)) {
        setError(OTA_ERROR_VERIFICATION, "Hash verification failed");
        reportStatus(update.releaseID, OTA_STATUS_FAILED, 0, "Hash verification failed");
        goto cleanup;
    }
    
    // Verify signature if enabled
    if (_verifySignature && update.signature.length() > 0) {
        if (!verifySignature(firmwareData, downloadedSize, update.signature)) {
            setError(OTA_ERROR_VERIFICATION, "Signature verification failed");
            reportStatus(update.releaseID, OTA_STATUS_FAILED, 0, "Signature verification failed");
            goto cleanup;
        }
    }
    
    // Report installing status
    reportStatus(update.releaseID, OTA_STATUS_INSTALLING, 50);
    if (_statusCallback) {
        _statusCallback(OTA_STATUS_INSTALLING, 50);
    }
    
    // Install firmware
    if (!installFirmware(firmwareData, downloadedSize)) {
        reportStatus(update.releaseID, OTA_STATUS_FAILED, 50, _lastErrorMessage.c_str());
        goto cleanup;
    }
    
    // Report completed status
    reportStatus(update.releaseID, OTA_STATUS_COMPLETED, 100);
    if (_statusCallback) {
        _statusCallback(OTA_STATUS_COMPLETED, 100);
    }
    
    success = true;
    _lastError = OTA_ERROR_NONE;
    
cleanup:
    if (firmwareData) {
        free(firmwareData);
    }
    
    return success;
}

bool OTAClient::checkAndUpdate() {
    FirmwareUpdate update;
    
    if (!checkForUpdate(&update)) {
        return false;
    }
    
    return performUpdate(update);
}

void OTAClient::setProgressCallback(ProgressCallback callback) {
    _progressCallback = callback;
}

void OTAClient::setStatusCallback(StatusCallback callback) {
    _statusCallback = callback;
}

int OTAClient::getLastError() const {
    return _lastError;
}

const char* OTAClient::getLastErrorMessage() const {
    return _lastErrorMessage.c_str();
}

void OTAClient::setCACertificate(const char* caCert) {
    _caCert = String(caCert);
}

void OTAClient::setVerifySignature(bool enable) {
    _verifySignature = enable;
}

bool OTAClient::reportStatus(const String& releaseID, const char* status, int progress, const char* errorMessage) {
    String url = _serverURL + "/api/v1/ota/updates/status";
    
    _httpClient.begin(_wifiClient, url);
    _httpClient.addHeader("Content-Type", "application/json");
    
    // Build JSON payload
    DynamicJsonDocument doc(512);
    doc["device_id"] = _deviceID;
    doc["release_id"] = releaseID;
    doc["status"] = status;
    doc["progress"] = progress;
    if (errorMessage) {
        doc["error_message"] = errorMessage;
    }
    
    String payload;
    serializeJson(doc, payload);
    
    int httpCode = _httpClient.POST(payload);
    _httpClient.end();
    
    return (httpCode == HTTP_CODE_OK);
}

size_t OTAClient::downloadFirmware(const String& url, uint8_t** buffer, size_t expectedSize) {
    _httpClient.begin(_wifiClient, url);
    
    int httpCode = _httpClient.GET();
    
    if (httpCode != HTTP_CODE_OK) {
        _httpClient.end();
        setError(OTA_ERROR_DOWNLOAD, "Download failed: HTTP " + String(httpCode));
        return 0;
    }
    
    int contentLength = _httpClient.getSize();
    if (contentLength <= 0) {
        _httpClient.end();
        setError(OTA_ERROR_DOWNLOAD, "Invalid content length");
        return 0;
    }
    
    // Allocate buffer
    *buffer = (uint8_t*)malloc(contentLength);
    if (*buffer == nullptr) {
        _httpClient.end();
        setError(OTA_ERROR_DOWNLOAD, "Memory allocation failed");
        return 0;
    }
    
    // Download data
    WiFiClient* stream = _httpClient.getStreamPtr();
    size_t totalRead = 0;
    size_t bytesRead = 0;
    uint8_t buff[128];
    
    while (_httpClient.connected() && (contentLength > 0 || contentLength == -1)) {
        size_t available = stream->available();
        
        if (available) {
            bytesRead = stream->readBytes(buff, min(available, sizeof(buff)));
            memcpy(*buffer + totalRead, buff, bytesRead);
            totalRead += bytesRead;
            
            if (_progressCallback) {
                _progressCallback(totalRead, contentLength);
            }
            
            if (contentLength > 0) {
                contentLength -= bytesRead;
            }
        }
        
        delay(1);
    }
    
    _httpClient.end();
    
    if (totalRead != expectedSize) {
        setError(OTA_ERROR_DOWNLOAD, "Downloaded size mismatch");
        free(*buffer);
        *buffer = nullptr;
        return 0;
    }
    
    return totalRead;
}

bool OTAClient::verifyHash(const uint8_t* data, size_t size, const String& expectedHash) {
    uint8_t hash[32];
    computeSHA256(data, size, hash);
    
    // Convert hash to hex string
    char hashStr[65];
    for (int i = 0; i < 32; i++) {
        sprintf(hashStr + (i * 2), "%02x", hash[i]);
    }
    hashStr[64] = '\0';
    
    return (expectedHash.equalsIgnoreCase(hashStr));
}

bool OTAClient::verifySignature(const uint8_t* data, size_t size, const String& signature) {
    // Decode base64 signature
    uint8_t sigBytes[512];
    size_t sigLen = base64Decode(signature, sigBytes, sizeof(sigBytes));
    
    if (sigLen == 0) {
        return false;
    }
    
    // Compute hash of data
    uint8_t hash[32];
    computeSHA256(data, size, hash);
    
    // Verify signature using mbedtls
    mbedtls_pk_context pk;
    mbedtls_pk_init(&pk);
    
    int ret = mbedtls_pk_parse_public_key(&pk, (const unsigned char*)_publicKey.c_str(), 
                                          _publicKey.length() + 1);
    if (ret != 0) {
        mbedtls_pk_free(&pk);
        return false;
    }
    
    ret = mbedtls_pk_verify(&pk, MBEDTLS_MD_SHA256, hash, sizeof(hash), sigBytes, sigLen);
    
    mbedtls_pk_free(&pk);
    
    return (ret == 0);
}

bool OTAClient::installFirmware(const uint8_t* data, size_t size) {
    if (!Update.begin(size)) {
        setError(OTA_ERROR_INSTALLATION, "Update begin failed: " + String(Update.errorString()));
        return false;
    }
    
    size_t written = Update.write(data, size);
    if (written != size) {
        Update.abort();
        setError(OTA_ERROR_INSTALLATION, "Update write failed: " + String(Update.errorString()));
        return false;
    }
    
    if (!Update.end()) {
        setError(OTA_ERROR_INSTALLATION, "Update end failed: " + String(Update.errorString()));
        return false;
    }
    
    if (!Update.isFinished()) {
        setError(OTA_ERROR_INSTALLATION, "Update not finished");
        return false;
    }
    
    return true;
}

void OTAClient::setError(int errorCode, const String& errorMessage) {
    _lastError = errorCode;
    _lastErrorMessage = errorMessage;
}

void OTAClient::computeSHA256(const uint8_t* data, size_t size, uint8_t* output) {
    mbedtls_sha256_context ctx;
    mbedtls_sha256_init(&ctx);
    mbedtls_sha256_starts(&ctx, 0); // 0 = SHA-256 (not SHA-224)
    mbedtls_sha256_update(&ctx, data, size);
    mbedtls_sha256_finish(&ctx, output);
    mbedtls_sha256_free(&ctx);
}

size_t OTAClient::hexToBytes(const String& hex, uint8_t* bytes, size_t maxLen) {
    size_t len = hex.length() / 2;
    if (len > maxLen) {
        len = maxLen;
    }
    
    for (size_t i = 0; i < len; i++) {
        sscanf(hex.substring(i * 2, i * 2 + 2).c_str(), "%02x", &bytes[i]);
    }
    
    return len;
}

size_t OTAClient::base64Decode(const String& input, uint8_t* output, size_t maxLen) {
    return base64_decode((char*)output, (char*)input.c_str(), input.length());
}
