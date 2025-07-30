//
//  device_enumerator.m
//  Device Enumeration - Audio & MIDI Device Discovery
//

#import "device_enumerator.h"
#import <Foundation/Foundation.h>
#import <CoreAudio/CoreAudio.h>
#import <AudioToolbox/AudioToolbox.h>
#import <CoreMIDI/CoreMIDI.h>

static NSString* createDeviceJSON(NSArray* devices) {
    NSError *error = nil;
    NSData *jsonData = [NSJSONSerialization dataWithJSONObject:devices 
                                                       options:NSJSONWritingPrettyPrinted 
                                                         error:&error];
    if (error) {
        NSLog(@"JSON serialization error: %@", error);
        return @"[]";
    }
    return [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
}

char* enumerateAudioInputDevices(void) {
    @autoreleasepool {
        NSMutableArray *inputDevices = [[NSMutableArray alloc] init];
        
        // Get all audio devices
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        if (status != noErr) {
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            return strdup("[]");
        }
        
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        if (status != noErr) {
            free(deviceIDs);
            return strdup("[]");
        }
        
        // Check each device for input capabilities
        for (UInt32 i = 0; i < deviceCount; i++) {
            AudioDeviceID deviceID = deviceIDs[i];
            
            // Check input stream configuration
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeInput;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status != noErr) continue;
            
            AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            
            UInt32 inputChannels = 0;
            if (status == noErr) {
                for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                    inputChannels += bufferList->mBuffers[j].mNumberChannels;
                }
            }
            free(bufferList);
            
            if (inputChannels > 0) {
                // Get device name
                CFStringRef deviceName = NULL;
                propertyAddress.mSelector = kAudioDevicePropertyDeviceNameCFString;
                propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
                dataSize = sizeof(CFStringRef);
                
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceName);
                
                NSString *name = @"Unknown Device";
                if (status == noErr && deviceName) {
                    name = (__bridge NSString *)deviceName;
                }
                
                // Get device UID
                CFStringRef deviceUID = NULL;
                propertyAddress.mSelector = kAudioDevicePropertyDeviceUID;
                dataSize = sizeof(CFStringRef);
                
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceUID);
                
                NSString *uid = [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID];
                if (status == noErr && deviceUID) {
                    uid = (__bridge NSString *)deviceUID;
                }
                
                NSDictionary *device = @{
                    @"name": name,
                    @"uid": uid, 
                    @"deviceId": @(deviceID),
                    @"channelCount": @(inputChannels),
                    @"supportedSampleRates": @[@44100.0, @48000.0],
                    @"supportedBitDepths": @[@16, @24],
                    @"isDefault": @NO
                };
                
                [inputDevices addObject:device];
                
                if (deviceName) CFRelease(deviceName);
                if (deviceUID) CFRelease(deviceUID);
            }
        }
        
        free(deviceIDs);
        
        NSString *jsonString = createDeviceJSON(inputDevices);
        return strdup([jsonString UTF8String]);
    }
}

char* enumerateAudioOutputDevices(void) {
    @autoreleasepool {
        NSMutableArray *outputDevices = [[NSMutableArray alloc] init];
        
        // Get all audio devices
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        if (status != noErr) {
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            return strdup("[]");
        }
        
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        if (status != noErr) {
            free(deviceIDs);
            return strdup("[]");
        }
        
        // Check each device for output capabilities
        for (UInt32 i = 0; i < deviceCount; i++) {
            AudioDeviceID deviceID = deviceIDs[i];
            
            // Check output stream configuration
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeOutput;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status != noErr) continue;
            
            AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            
            UInt32 outputChannels = 0;
            if (status == noErr) {
                for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                    outputChannels += bufferList->mBuffers[j].mNumberChannels;
                }
            }
            free(bufferList);
            
            if (outputChannels > 0) {
                // Get device name
                CFStringRef deviceName = NULL;
                propertyAddress.mSelector = kAudioDevicePropertyDeviceNameCFString;
                propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
                dataSize = sizeof(CFStringRef);
                
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceName);
                
                NSString *name = @"Unknown Device";
                if (status == noErr && deviceName) {
                    name = (__bridge NSString *)deviceName;
                }
                
                // Get device UID
                CFStringRef deviceUID = NULL;
                propertyAddress.mSelector = kAudioDevicePropertyDeviceUID;
                dataSize = sizeof(CFStringRef);
                
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceUID);
                
                NSString *uid = [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID];
                if (status == noErr && deviceUID) {
                    uid = (__bridge NSString *)deviceUID;
                }
                
                NSDictionary *device = @{
                    @"name": name,
                    @"uid": uid,
                    @"deviceId": @(deviceID),
                    @"channelCount": @(outputChannels),
                    @"supportedSampleRates": @[@44100.0, @48000.0],
                    @"supportedBitDepths": @[@16, @24],
                    @"isDefault": @NO
                };
                
                [outputDevices addObject:device];
                
                if (deviceName) CFRelease(deviceName);
                if (deviceUID) CFRelease(deviceUID);
            }
        }
        
        free(deviceIDs);
        
        NSString *jsonString = createDeviceJSON(outputDevices);
        return strdup([jsonString UTF8String]);
    }
}

char* enumerateMIDIInputDevices(void) {
    @autoreleasepool {
        NSMutableArray *midiInputs = [[NSMutableArray alloc] init];
        
        ItemCount sourceCount = MIDIGetNumberOfSources();
        
        for (ItemCount i = 0; i < sourceCount; i++) {
            MIDIEndpointRef source = MIDIGetSource(i);
            if (source == 0) continue;
            
            CFStringRef name = NULL;
            OSStatus status = MIDIObjectGetStringProperty(source, kMIDIPropertyName, &name);
            
            NSString *deviceName = @"Unknown MIDI Input";
            if (status == noErr && name) {
                deviceName = (__bridge NSString *)name;
            }
            
            CFStringRef uid = NULL;
            status = MIDIObjectGetStringProperty(source, kMIDIPropertyUniqueID, &uid);
            
            NSString *deviceUID = [NSString stringWithFormat:@"midi_input_%lu", (unsigned long)i];
            if (status == noErr && uid) {
                deviceUID = (__bridge NSString *)uid;
            }
            
            NSDictionary *device = @{
                @"name": deviceName,
                @"uid": deviceUID,
                @"endpointId": @((int)source),
                @"isOnline": @YES
            };
            
            [midiInputs addObject:device];
            
            if (name) CFRelease(name);
            if (uid) CFRelease(uid);
        }
        
        NSString *jsonString = createDeviceJSON(midiInputs);
        return strdup([jsonString UTF8String]);
    }
}

char* enumerateMIDIOutputDevices(void) {
    @autoreleasepool {
        NSMutableArray *midiOutputs = [[NSMutableArray alloc] init];
        
        ItemCount destCount = MIDIGetNumberOfDestinations();
        
        for (ItemCount i = 0; i < destCount; i++) {
            MIDIEndpointRef dest = MIDIGetDestination(i);
            if (dest == 0) continue;
            
            CFStringRef name = NULL;
            OSStatus status = MIDIObjectGetStringProperty(dest, kMIDIPropertyName, &name);
            
            NSString *deviceName = @"Unknown MIDI Output";
            if (status == noErr && name) {
                deviceName = (__bridge NSString *)name;
            }
            
            CFStringRef uid = NULL;
            status = MIDIObjectGetStringProperty(dest, kMIDIPropertyUniqueID, &uid);
            
            NSString *deviceUID = [NSString stringWithFormat:@"midi_output_%lu", (unsigned long)i];
            if (status == noErr && uid) {
                deviceUID = (__bridge NSString *)uid;
            }
            
            NSDictionary *device = @{
                @"name": deviceName,
                @"uid": deviceUID,
                @"endpointId": @((int)dest),
                @"isOnline": @YES
            };
            
            [midiOutputs addObject:device];
            
            if (name) CFRelease(name);
            if (uid) CFRelease(uid);
        }
        
        NSString *jsonString = createDeviceJSON(midiOutputs);
        return strdup([jsonString UTF8String]);
    }
}

char* getDefaultAudioDevices(void) {
    @autoreleasepool {
        AudioObjectPropertyAddress propertyAddress;
        UInt32 dataSize = sizeof(AudioDeviceID);
        AudioDeviceID defaultInput = 0, defaultOutput = 0;
        
        // Get default input device
        propertyAddress.mSelector = kAudioHardwarePropertyDefaultInputDevice;
        propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
        propertyAddress.mElement = kAudioObjectPropertyElementMain;
        
        AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, &defaultInput);
        
        // Get default output device
        propertyAddress.mSelector = kAudioHardwarePropertyDefaultOutputDevice;
        
        AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, &defaultOutput);
        
        NSDictionary *defaults = @{
            @"defaultInput": @(defaultInput),
            @"defaultOutput": @(defaultOutput)
        };
        
        NSError *error = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:defaults 
                                                           options:0 
                                                             error:&error];
        if (error) {
            return strdup("{}");
        }
        
        NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([jsonString UTF8String]);
    }
}

char* enumerateAllDevices(void) {
    @autoreleasepool {
        // Get all device types
        char* audioInputsJSON = enumerateAudioInputDevices();
        char* audioOutputsJSON = enumerateAudioOutputDevices();
        char* midiInputsJSON = enumerateMIDIInputDevices();
        char* midiOutputsJSON = enumerateMIDIOutputDevices();
        char* defaultsJSON = getDefaultAudioDevices();
        
        // Parse JSON strings back to objects
        NSData *audioInputsData = [[NSString stringWithUTF8String:audioInputsJSON] dataUsingEncoding:NSUTF8StringEncoding];
        NSData *audioOutputsData = [[NSString stringWithUTF8String:audioOutputsJSON] dataUsingEncoding:NSUTF8StringEncoding];
        NSData *midiInputsData = [[NSString stringWithUTF8String:midiInputsJSON] dataUsingEncoding:NSUTF8StringEncoding];
        NSData *midiOutputsData = [[NSString stringWithUTF8String:midiOutputsJSON] dataUsingEncoding:NSUTF8StringEncoding];
        NSData *defaultsData = [[NSString stringWithUTF8String:defaultsJSON] dataUsingEncoding:NSUTF8StringEncoding];
        
        NSArray *audioInputs = [NSJSONSerialization JSONObjectWithData:audioInputsData options:0 error:nil] ?: @[];
        NSArray *audioOutputs = [NSJSONSerialization JSONObjectWithData:audioOutputsData options:0 error:nil] ?: @[];
        NSArray *midiInputs = [NSJSONSerialization JSONObjectWithData:midiInputsData options:0 error:nil] ?: @[];
        NSArray *midiOutputs = [NSJSONSerialization JSONObjectWithData:midiOutputsData options:0 error:nil] ?: @[];
        NSDictionary *defaults = [NSJSONSerialization JSONObjectWithData:defaultsData options:0 error:nil] ?: @{};
        
        // Add "(None Selected)" options
        NSMutableArray *allAudioInputs = [audioInputs mutableCopy];
        [allAudioInputs insertObject:@{@"name": @"(None Selected)", @"uid": @"none", @"deviceId": @-1, @"channelCount": @0, @"supportedSampleRates": @[], @"supportedBitDepths": @[], @"isDefault": @NO} atIndex:0];
        
        NSMutableArray *allAudioOutputs = [audioOutputs mutableCopy];
        [allAudioOutputs insertObject:@{@"name": @"(None Selected)", @"uid": @"none", @"deviceId": @-1, @"channelCount": @0, @"supportedSampleRates": @[], @"supportedBitDepths": @[], @"isDefault": @NO} atIndex:0];
        
        NSMutableArray *allMIDIInputs = [midiInputs mutableCopy];
        [allMIDIInputs insertObject:@{@"name": @"(None Selected)", @"uid": @"none", @"endpointId": @-1, @"isOnline": @NO} atIndex:0];
        
        NSMutableArray *allMIDIOutputs = [midiOutputs mutableCopy];
        [allMIDIOutputs insertObject:@{@"name": @"(None Selected)", @"uid": @"none", @"endpointId": @-1, @"isOnline": @NO} atIndex:0];
        
        NSDictionary *result = @{
            @"audioInputs": allAudioInputs,
            @"audioOutputs": allAudioOutputs,
            @"midiInputs": allMIDIInputs,
            @"midiOutputs": allMIDIOutputs,
            @"defaultDevices": defaults,
            @"success": @YES,
            @"error": @""
        };
        
        // Clean up allocated strings
        free(audioInputsJSON);
        free(audioOutputsJSON);
        free(midiInputsJSON);
        free(midiOutputsJSON);
        free(defaultsJSON);
        
        NSError *error = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:result 
                                                           options:NSJSONWritingPrettyPrinted 
                                                             error:&error];
        if (error) {
            return strdup("{}");
        }
        
        NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([jsonString UTF8String]);
    }
}
