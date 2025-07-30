//
//  audiounit_devices.h
//  AudioUnit device enumeration for rackless
//  Extracted from Archive and optimized for production use
//

#ifndef audiounit_devices_h
#define audiounit_devices_h

#ifdef __cplusplus
extern "C" {
#endif

// Audio device enumeration functions
char* getAudioInputDevices(void);
char* getAudioOutputDevices(void);
char* getDefaultAudioDevices(void);

// MIDI device enumeration functions  
char* getMIDIInputDevices(void);
char* getMIDIOutputDevices(void);

// Utility functions
int getAudioDeviceCount(int isInput);
int getMIDIDeviceCount(int isInput);

#ifdef __cplusplus
}
#endif

#endif /* audiounit_devices_h */
