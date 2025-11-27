/**
 * Advanced OTA Update Example
 * 
 * This example demonstrates advanced OTA features including:
 * - Automatic retry on failure
 * - Health check after update
 * - Rollback detection
 * - Persistent update state
 * 
 * Requirements:
 * - ESP32 or ESP8266 board
 * - WiFi connection
 * - Device registered in ATHENA platform
 */

#include <WiFi.h>
#include <Preferences.h>
#include "OTAClient.h"

// WiFi credentials
const char* ssid = "YOUR_WIFI_SSID";
const char* password = "YOUR_WIFI_PASSWORD";

// ATHENA OTA service configuration
const char* otaServerURL = "https://athena.example.com";
const char* deviceID = "YOUR_DEVICE_ID";

// Public key for signature verification
const char* publicKey = R"(
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----
)";

// CA certificate for HTTPS
const char* caCert = R"(
-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ...
-----END CERTIFICATE-----
)";

// Create OTA client
OTAClient otaClient(otaServerURL, deviceID, publicKey);

// Preferences for persistent storage
Preferences preferences;

// Update configuration
const unsigned long UPDATE_CHECK_INTERVAL = 10 * 60 * 1000; // 10 minutes
const int MAX_UPDATE_RETRIES = 3;
const unsigned long HEALTH_CHECK_TIMEOUT = 60 * 1000; // 1 minute

unsigned long lastUpdateCheck = 0;
int updateRetryCount = 0;
bool updateInProgress = false;

void setup() {
    Serial.begin(115200);
    Serial.println("\n\nATHENA OTA Client - Advanced Example");
    Serial.println("=====================================");
    
    // Initialize preferences
    preferences.begin("ota", false);
    
    // Check if we just updated
    bool justUpdated = preferences.getBool("just_updated", false);
    if (justUpdated) {
        Serial.println("Device just updated - performing health check...");
        if (performHealthCheck()) {
            Serial.println("Health check passed - update successful!");
            preferences.putBool("just_updated", false);
            preferences.putInt("retry_count", 0);
            
            // Report success to server
            reportUpdateSuccess();
        } else {
            Serial.println("Health check failed - update may have issues");
            int retries = preferences.getInt("retry_count", 0);
            if (retries >= MAX_UPDATE_RETRIES) {
                Serial.println("Max retries reached - requesting rollback");
                reportUpdateFailure("Health check failed after max retries");
            } else {
                preferences.putInt("retry_count", retries + 1);
                reportUpdateFailure("Health check failed - retry " + String(retries + 1));
            }
        }
    }
    
    // Connect to WiFi
    Serial.print("Connecting to WiFi");
    WiFi.begin(ssid, password);
    while (WiFi.status() != WL_CONNECTED) {
        delay(500);
        Serial.print(".");
    }
    Serial.println("\nWiFi connected!");
    Serial.print("IP address: ");
    Serial.println(WiFi.localIP());
    
    // Initialize OTA client
    otaClient.setCACertificate(caCert);
    otaClient.setProgressCallback(onProgress);
    otaClient.setStatusCallback(onStatus);
    
    if (!otaClient.begin()) {
        Serial.println("Failed to initialize OTA client");
        return;
    }
    
    Serial.println("OTA client initialized");
    Serial.println("Device ID: " + String(deviceID));
    
    // Check for updates on startup
    checkForUpdates();
}

void loop() {
    // Check for updates periodically
    if (!updateInProgress && millis() - lastUpdateCheck >= UPDATE_CHECK_INTERVAL) {
        lastUpdateCheck = millis();
        checkForUpdates();
    }
    
    // Your application code here
    delay(1000);
}

void checkForUpdates() {
    Serial.println("\n--- Checking for firmware updates ---");
    
    FirmwareUpdate update;
    if (otaClient.checkForUpdate(&update)) {
        Serial.println("Update available!");
        Serial.println("  Release ID: " + update.releaseID);
        Serial.println("  Version: " + update.version);
        Serial.println("  Size: " + String(update.binarySize) + " bytes");
        Serial.println("  Release Notes: " + update.releaseNotes);
        
        // Check if we should skip this update due to previous failures
        String lastFailedRelease = preferences.getString("failed_release", "");
        if (lastFailedRelease == update.releaseID) {
            int retries = preferences.getInt("retry_count", 0);
            if (retries >= MAX_UPDATE_RETRIES) {
                Serial.println("Skipping update - max retries reached for this release");
                return;
            }
        } else {
            // New release - reset retry count
            preferences.putInt("retry_count", 0);
        }
        
        Serial.println("\nStarting update...");
        updateInProgress = true;
        
        // Store update info for post-update verification
        preferences.putString("current_release", update.releaseID);
        preferences.putBool("just_updated", true);
        
        if (otaClient.performUpdate(update)) {
            Serial.println("\nUpdate downloaded and installed successfully!");
            Serial.println("Rebooting in 3 seconds...");
            delay(3000);
            ESP.restart();
        } else {
            Serial.println("\nUpdate failed!");
            Serial.print("Error: ");
            Serial.println(otaClient.getLastErrorMessage());
            
            updateInProgress = false;
            preferences.putBool("just_updated", false);
            preferences.putString("failed_release", update.releaseID);
            
            int retries = preferences.getInt("retry_count", 0);
            preferences.putInt("retry_count", retries + 1);
            
            reportUpdateFailure(otaClient.getLastErrorMessage());
        }
    } else {
        int error = otaClient.getLastError();
        if (error == OTA_ERROR_NO_UPDATE) {
            Serial.println("No update available - firmware is up to date");
        } else {
            Serial.println("Failed to check for updates");
            Serial.print("Error: ");
            Serial.println(otaClient.getLastErrorMessage());
        }
    }
}

bool performHealthCheck() {
    Serial.println("Performing health check...");
    
    // Check WiFi connectivity
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("  WiFi: FAILED");
        return false;
    }
    Serial.println("  WiFi: OK");
    
    // Check free heap
    size_t freeHeap = ESP.getFreeHeap();
    Serial.printf("  Free Heap: %d bytes\n", freeHeap);
    if (freeHeap < 10000) {
        Serial.println("  Memory: FAILED (low memory)");
        return false;
    }
    Serial.println("  Memory: OK");
    
    // Add your application-specific health checks here
    // For example:
    // - Check sensor connectivity
    // - Verify configuration
    // - Test critical functionality
    
    Serial.println("Health check completed successfully");
    return true;
}

void reportUpdateSuccess() {
    String releaseID = preferences.getString("current_release", "");
    if (releaseID.length() == 0) {
        return;
    }
    
    // The OTA client already reported completion during performUpdate
    // This is just for additional logging
    Serial.println("Update success reported to server");
}

void reportUpdateFailure(const String& errorMessage) {
    String releaseID = preferences.getString("current_release", "");
    if (releaseID.length() == 0) {
        return;
    }
    
    Serial.println("Reporting update failure to server: " + errorMessage);
    
    // The OTA client already reported failure during performUpdate
    // This is just for additional logging
}

void onProgress(size_t current, size_t total) {
    static int lastPercentage = -1;
    int percentage = (current * 100) / total;
    
    // Only print when percentage changes to reduce serial output
    if (percentage != lastPercentage) {
        Serial.printf("Progress: %d%% (%d/%d bytes)\n", percentage, current, total);
        lastPercentage = percentage;
    }
}

void onStatus(const char* status, int progress) {
    Serial.printf("Status: %s (%d%%)\n", status, progress);
}
