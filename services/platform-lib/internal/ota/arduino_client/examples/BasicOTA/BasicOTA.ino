/**
 * Basic OTA Update Example
 * 
 * This example demonstrates how to use the OTAClient library to check for
 * and install firmware updates from the ATHENA OTA service.
 * 
 * Requirements:
 * - ESP32 or ESP8266 board
 * - WiFi connection
 * - Device registered in ATHENA platform
 */

#include <WiFi.h>
#include "OTAClient.h"

// WiFi credentials
const char* ssid = "YOUR_WIFI_SSID";
const char* password = "YOUR_WIFI_PASSWORD";

// ATHENA OTA service configuration
const char* otaServerURL = "https://athena.example.com";
const char* deviceID = "YOUR_DEVICE_ID";

// Public key for signature verification (PEM format)
const char* publicKey = R"(
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----
)";

// CA certificate for HTTPS (optional but recommended)
const char* caCert = R"(
-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ...
-----END CERTIFICATE-----
)";

// Create OTA client
OTAClient otaClient(otaServerURL, deviceID, publicKey);

// Check for updates every 5 minutes
const unsigned long UPDATE_CHECK_INTERVAL = 5 * 60 * 1000;
unsigned long lastUpdateCheck = 0;

void setup() {
    Serial.begin(115200);
    Serial.println("\n\nATHENA OTA Client - Basic Example");
    Serial.println("==================================");
    
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
    Serial.println("\nChecking for updates...");
    
    // Check for updates immediately on startup
    checkForUpdates();
}

void loop() {
    // Check for updates periodically
    if (millis() - lastUpdateCheck >= UPDATE_CHECK_INTERVAL) {
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
        Serial.println("\nStarting update...");
        
        if (otaClient.performUpdate(update)) {
            Serial.println("\nUpdate completed successfully!");
            Serial.println("Rebooting in 3 seconds...");
            delay(3000);
            ESP.restart();
        } else {
            Serial.println("\nUpdate failed!");
            Serial.print("Error: ");
            Serial.println(otaClient.getLastErrorMessage());
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

void onProgress(size_t current, size_t total) {
    int percentage = (current * 100) / total;
    Serial.printf("Progress: %d%% (%d/%d bytes)\n", percentage, current, total);
}

void onStatus(const char* status, int progress) {
    Serial.printf("Status: %s (%d%%)\n", status, progress);
}
