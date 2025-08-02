//
//  audiounit_devices.h
//  Rackless - Audio & MIDI Device Enumeration
//
//  Provides functions for discovering available audio input/output devices
//  and MIDI input/output devices on the system.
//

#ifndef audiounit_devices_h
#define audiounit_devices_h

#ifdef __cplusplus
extern "C" {
#endif

// Device information structure
typedef struct {
    char name[256];
    char uid[256];
    int deviceId;
    int isDefault;
    int inputChannels;
    int outputChannels;
} AudioDeviceInfo;

typedef struct {
    char name[256];
    char uid[256];
    int endpointId;
    int isOnline;
} MIDIDeviceInfo;

// Audio device enumeration
char* getAudioInputDevices(void);
char* getAudioOutputDevices(void);
char* getDefaultAudioDevices(void);

// MIDI device enumeration  
char* getMIDIInputDevices(void);
char* getMIDIOutputDevices(void);

// Audio configuration
double getDefaultSampleRate(void);

// Utility functions
int getAudioDeviceCount(int isInput);
int getMIDIDeviceCount(int isInput);

#ifdef __cplusplus
}
#endif

#endif /* audiounit_devices_h */
