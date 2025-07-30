//
//  device_enumerator.h 
//  Device Enumeration - Audio & MIDI Device Discovery
//
//  Provides comprehensive device enumeration for the rackless audio system
//

#ifndef device_enumerator_h
#define device_enumerator_h

#ifdef __cplusplus
extern "C" {
#endif

// Device enumeration functions
char* enumerateAudioInputDevices(void);
char* enumerateAudioOutputDevices(void); 
char* enumerateMIDIInputDevices(void);
char* enumerateMIDIOutputDevices(void);
char* getDefaultAudioDevices(void);
char* enumerateAllDevices(void);

#ifdef __cplusplus
}
#endif

#endif /* device_enumerator_h */
