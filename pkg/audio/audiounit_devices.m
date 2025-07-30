//
//  audiounit_devices.m
//  AudioUnit device enumeration for rackless
//  Extracted from Archive with timeout optimization and production-ready error handling
//

#import "audiounit_devices.h"
#import <Foundation/Foundation.h>
#import <CoreAudio/CoreAudio.h>
#import <CoreMIDI/CoreMIDI.h>

// Audio Input Device Enumeration
char* getAudioInputDevices(void) {
    @autoreleasepool {
        NSLog(@"🎤 Enumerating audio input devices...");
        
        // Get all audio devices
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get audio device count: %d", (int)status);
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            NSLog(@"❌ Failed to allocate memory for device IDs");
            return strdup("[]");
        }
        
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        if (status != noErr) {
            NSLog(@"❌ Failed to get device IDs: %d", (int)status);
            free(deviceIDs);
            return strdup("[]");
        }
        
        NSMutableArray *inputDevices = [[NSMutableArray alloc] init];
        
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
            
            if (inputChannels == 0) continue; // Skip devices without input channels
            
            // Get device name
            propertyAddress.mSelector = kAudioDevicePropertyDeviceName;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            dataSize = 256;
            char deviceName[256];
            
            NSString *name = @"Unknown Device";
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, deviceName);
            if (status == noErr) {
                name = [NSString stringWithUTF8String:deviceName];
            }
            
            // Get supported sample rates
            NSMutableArray *sampleRates = [[NSMutableArray alloc] init];
            propertyAddress.mSelector = kAudioDevicePropertyAvailableNominalSampleRates;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioValueRange *sampleRateRanges = (AudioValueRange *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, sampleRateRanges);
                
                if (status == noErr) {
                    UInt32 rangeCount = dataSize / sizeof(AudioValueRange);
                    for (UInt32 r = 0; r < rangeCount; r++) {
                        double minRate = sampleRateRanges[r].mMinimum;
                        double maxRate = sampleRateRanges[r].mMaximum;
                        
                        // Add common sample rates within this range
                        double commonRates[] = {44100, 48000, 88200, 96000, 176400, 192000};
                        for (int cr = 0; cr < 6; cr++) {
                            if (commonRates[cr] >= minRate && commonRates[cr] <= maxRate) {
                                [sampleRates addObject:@(commonRates[cr])];
                            }
                        }
                    }
                }
                free(sampleRateRanges);
            } else {
                // Default sample rates if not available
                [sampleRates addObjectsFromArray:@[@44100, @48000]];
            }
            
            // Get supported bit depths
            NSMutableArray *bitDepths = [[NSMutableArray alloc] init];
            propertyAddress.mSelector = kAudioDevicePropertyStreamFormats;
            propertyAddress.mScope = kAudioDevicePropertyScopeInput;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioStreamBasicDescription *formats = (AudioStreamBasicDescription *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, formats);
                
                if (status == noErr) {
                    UInt32 formatCount = dataSize / sizeof(AudioStreamBasicDescription);
                    NSMutableSet *uniqueBitDepths = [[NSMutableSet alloc] init];
                    
                    for (UInt32 f = 0; f < formatCount; f++) {
                        UInt32 bitsPerChannel = formats[f].mBitsPerChannel;
                        if (bitsPerChannel > 0) {
                            [uniqueBitDepths addObject:@(bitsPerChannel)];
                        }
                    }
                    [bitDepths addObjectsFromArray:[uniqueBitDepths allObjects]];
                }
                free(formats);
            } else {
                // Default bit depths if not available
                [bitDepths addObjectsFromArray:@[@16, @24]];
            }
            
            // Create device info
            [inputDevices addObject:@{
                @"name": name,
                @"uid": [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID],
                @"deviceId": @(deviceID),
                @"channelCount": @(inputChannels),
                @"supportedSampleRates": sampleRates,
                @"supportedBitDepths": bitDepths,
                @"isDefault": @NO
            }];
        }
        
        free(deviceIDs);
        
        NSLog(@"✅ Found %lu audio input devices", (unsigned long)[inputDevices count]);
        
        // Convert to JSON
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:inputDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}

// Audio Output Device Enumeration
char* getAudioOutputDevices(void) {
    @autoreleasepool {
        NSLog(@"🔊 Enumerating audio output devices...");
        
        // Get all audio devices
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get audio device count: %d", (int)status);
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            NSLog(@"❌ Failed to allocate memory for device IDs");
            return strdup("[]");
        }
        
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        if (status != noErr) {
            NSLog(@"❌ Failed to get device IDs: %d", (int)status);
            free(deviceIDs);
            return strdup("[]");
        }
        
        NSMutableArray *outputDevices = [[NSMutableArray alloc] init];
        
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
            
            if (outputChannels == 0) continue; // Skip devices without output channels
            
            // Get device name
            propertyAddress.mSelector = kAudioDevicePropertyDeviceName;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            dataSize = 256;
            char deviceName[256];
            
            NSString *name = @"Unknown Device";
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, deviceName);
            if (status == noErr) {
                name = [NSString stringWithUTF8String:deviceName];
            }
            
            // Get supported sample rates
            NSMutableArray *sampleRates = [[NSMutableArray alloc] init];
            propertyAddress.mSelector = kAudioDevicePropertyAvailableNominalSampleRates;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioValueRange *sampleRateRanges = (AudioValueRange *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, sampleRateRanges);
                
                if (status == noErr) {
                    UInt32 rangeCount = dataSize / sizeof(AudioValueRange);
                    for (UInt32 r = 0; r < rangeCount; r++) {
                        double minRate = sampleRateRanges[r].mMinimum;
                        double maxRate = sampleRateRanges[r].mMaximum;
                        
                        // Add common sample rates within this range
                        double commonRates[] = {44100, 48000, 88200, 96000, 176400, 192000};
                        for (int cr = 0; cr < 6; cr++) {
                            if (commonRates[cr] >= minRate && commonRates[cr] <= maxRate) {
                                [sampleRates addObject:@(commonRates[cr])];
                            }
                        }
                    }
                }
                free(sampleRateRanges);
            } else {
                // Default sample rates if not available
                [sampleRates addObjectsFromArray:@[@44100, @48000]];
            }
            
            // Get supported bit depths
            NSMutableArray *bitDepths = [[NSMutableArray alloc] init];
            propertyAddress.mSelector = kAudioDevicePropertyStreamFormats;
            propertyAddress.mScope = kAudioDevicePropertyScopeOutput;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioStreamBasicDescription *formats = (AudioStreamBasicDescription *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, formats);
                
                if (status == noErr) {
                    UInt32 formatCount = dataSize / sizeof(AudioStreamBasicDescription);
                    NSMutableSet *uniqueBitDepths = [[NSMutableSet alloc] init];
                    
                    for (UInt32 f = 0; f < formatCount; f++) {
                        UInt32 bitsPerChannel = formats[f].mBitsPerChannel;
                        if (bitsPerChannel > 0) {
                            [uniqueBitDepths addObject:@(bitsPerChannel)];
                        }
                    }
                    [bitDepths addObjectsFromArray:[uniqueBitDepths allObjects]];
                }
                free(formats);
            } else {
                // Default bit depths if not available
                [bitDepths addObjectsFromArray:@[@16, @24]];
            }
            
            // Create device info
            [outputDevices addObject:@{
                @"name": name,
                @"uid": [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID],
                @"deviceId": @(deviceID),
                @"channelCount": @(outputChannels),
                @"supportedSampleRates": sampleRates,
                @"supportedBitDepths": bitDepths,
                @"isDefault": @NO
            }];
        }
        
        free(deviceIDs);
        
        NSLog(@"✅ Found %lu audio output devices", (unsigned long)[outputDevices count]);
        
        // Convert to JSON
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:outputDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}

// Default Audio Devices
char* getDefaultAudioDevices(void) {
    @autoreleasepool {
        NSLog(@"🎯 Getting default audio devices...");
        
        // Get system default output device
        AudioObjectPropertyAddress defaultOutputAddress = {
            kAudioHardwarePropertyDefaultOutputDevice,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        AudioDeviceID defaultOutputDeviceID = kAudioObjectUnknown;
        UInt32 size = sizeof(AudioDeviceID);
        OSStatus status = AudioObjectGetPropertyData(
            kAudioObjectSystemObject, 
            &defaultOutputAddress, 
            0, NULL, 
            &size, 
            &defaultOutputDeviceID
        );
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get default output device: %d", (int)status);
            defaultOutputDeviceID = kAudioObjectUnknown;
        }
        
        // Get system default input device  
        AudioObjectPropertyAddress defaultInputAddress = {
            kAudioHardwarePropertyDefaultInputDevice,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        AudioDeviceID defaultInputDeviceID = kAudioObjectUnknown;
        status = AudioObjectGetPropertyData(
            kAudioObjectSystemObject, 
            &defaultInputAddress, 
            0, NULL, 
            &size, 
            &defaultInputDeviceID
        );
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get default input device: %d", (int)status);
            defaultInputDeviceID = kAudioObjectUnknown;
        }
        
        NSLog(@"✅ Default input: %u, output: %u", (unsigned int)defaultInputDeviceID, (unsigned int)defaultOutputDeviceID);
        
        // Create JSON response
        NSDictionary *result = @{
            @"defaultInput": @(defaultInputDeviceID),
            @"defaultOutput": @(defaultOutputDeviceID)
        };
        
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:result options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ Default devices JSON serialization failed");
            return strdup("{\"defaultInput\":0,\"defaultOutput\":0}");
        }
        
        NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([jsonString UTF8String]);
    }
}

// MIDI Input Device Enumeration
char* getMIDIInputDevices(void) {
    @autoreleasepool {
        NSLog(@"🎹 Enumerating MIDI input devices...");
        
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        if (deviceCount == 0) {
            NSLog(@"🔇 No MIDI devices found");
            return strdup("[]");
        }
        
        NSMutableArray *inputDevices = [[NSMutableArray alloc] init];
        
        for (ItemCount i = 0; i < deviceCount; i++) {
            MIDIDeviceRef device = MIDIGetDevice(i);
            if (device == 0) continue;
            
            // Get device name
            CFStringRef deviceName;
            OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyName, &deviceName);
            if (status != noErr) continue;
            
            NSString *deviceNameString = (__bridge NSString *)deviceName;
            
            // Get device unique ID
            SInt32 uniqueID;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyUniqueID, &uniqueID);
            if (status != noErr) {
                CFRelease(deviceName);
                continue;
            }
            
            // Check if device is online
            SInt32 isOffline;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyOffline, &isOffline);
            BOOL online = (status != noErr) ? YES : !isOffline;
            
            // Get entities and input endpoints
            ItemCount entityCount = MIDIDeviceGetNumberOfEntities(device);
            for (ItemCount j = 0; j < entityCount; j++) {
                MIDIEntityRef entity = MIDIDeviceGetEntity(device, j);
                if (entity == 0) continue;
                
                // Get input endpoints (sources)
                ItemCount sourceCount = MIDIEntityGetNumberOfSources(entity);
                for (ItemCount k = 0; k < sourceCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetSource(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name (might be different from device name)
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    // Add to results
                    [inputDevices addObject:@{
                        @"name": finalName,
                        @"uid": [NSString stringWithFormat:@"midi_%d", uniqueID],
                        @"endpointId": @(endpoint),
                        @"isOnline": @(online)
                    }];
                    
                    if (endpointName) CFRelease(endpointName);
                }
            }
            
            CFRelease(deviceName);
        }
        
        NSLog(@"✅ Found %lu MIDI input endpoints", (unsigned long)[inputDevices count]);
        
        // Convert to JSON
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:inputDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ MIDI input JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}

// MIDI Output Device Enumeration
char* getMIDIOutputDevices(void) {
    @autoreleasepool {
        NSLog(@"🎹 Enumerating MIDI output devices...");
        
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        if (deviceCount == 0) {
            NSLog(@"🔇 No MIDI devices found");
            return strdup("[]");
        }
        
        NSMutableArray *outputDevices = [[NSMutableArray alloc] init];
        
        for (ItemCount i = 0; i < deviceCount; i++) {
            MIDIDeviceRef device = MIDIGetDevice(i);
            if (device == 0) continue;
            
            // Get device name
            CFStringRef deviceName;
            OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyName, &deviceName);
            if (status != noErr) continue;
            
            NSString *deviceNameString = (__bridge NSString *)deviceName;
            
            // Get device unique ID
            SInt32 uniqueID;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyUniqueID, &uniqueID);
            if (status != noErr) {
                CFRelease(deviceName);
                continue;
            }
            
            // Check if device is online
            SInt32 isOffline;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyOffline, &isOffline);
            BOOL online = (status != noErr) ? YES : !isOffline;
            
            // Get entities and output endpoints
            ItemCount entityCount = MIDIDeviceGetNumberOfEntities(device);
            for (ItemCount j = 0; j < entityCount; j++) {
                MIDIEntityRef entity = MIDIDeviceGetEntity(device, j);
                if (entity == 0) continue;
                
                // Get output endpoints (destinations)
                ItemCount destCount = MIDIEntityGetNumberOfDestinations(entity);
                for (ItemCount k = 0; k < destCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetDestination(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name (might be different from device name)
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    // Add to results
                    [outputDevices addObject:@{
                        @"name": finalName,
                        @"uid": [NSString stringWithFormat:@"midi_%d", uniqueID],
                        @"endpointId": @(endpoint),
                        @"isOnline": @(online)
                    }];
                    
                    if (endpointName) CFRelease(endpointName);
                }
            }
            
            CFRelease(deviceName);
        }
        
        NSLog(@"✅ Found %lu MIDI output endpoints", (unsigned long)[outputDevices count]);
        
        // Convert to JSON
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:outputDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ MIDI output JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}

// Utility Functions
int getAudioDeviceCount(int isInput) {
    NSLog(@"🔍 getAudioDeviceCount called with isInput: %d", isInput);
    
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    
    UInt32 dataSize = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
    
    if (status != noErr) {
        return 0;
    }
    
    return (int)(dataSize / sizeof(AudioDeviceID));
}

int getMIDIDeviceCount(int isInput) {
    NSLog(@"🔍 getMIDIDeviceCount called with isInput: %d", isInput);
    return (int)MIDIGetNumberOfDevices();
}
