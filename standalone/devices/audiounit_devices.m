//
//  audiounit_devices_simple.m
//  Simple device enumeration test with logging
//

#import "audiounit_devices.h"
#import <Foundation/Foundation.h>
#import <CoreAudio/CoreAudio.h>
#import <CoreMIDI/CoreMIDI.h>

// Simple test implementation with logging
char* getAudioInputDevices(void) {
    @autoreleasepool {
        NSLog(@"🔍 getAudioInputDevices called");
        
        // Get device count - step 1
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        
        NSLog(@"🔍 Step 1 - AudioObjectGetPropertyDataSize result: %d, dataSize: %u", (int)status, (unsigned int)dataSize);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get device count: %d", (int)status);
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        NSLog(@"✅ Step 1 complete - Found %u audio devices", (unsigned int)deviceCount);
        
        // Get actual device IDs - step 2
        NSLog(@"🔍 Step 2 - Allocating memory for %u device IDs", (unsigned int)deviceCount);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            NSLog(@"❌ Failed to allocate memory for device IDs");
            return strdup("[]");
        }
        
        NSLog(@"🔍 Step 2 - Getting actual device IDs");
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get device IDs: %d", (int)status);
            free(deviceIDs);
            return strdup("[]");
        }
        
        NSLog(@"✅ Step 2 complete - Got device IDs successfully");
        
        // Log ALL device IDs to see the pattern
        NSLog(@"🔍 All %u device IDs:", (unsigned int)deviceCount);
        for (UInt32 i = 0; i < deviceCount; i++) {
            NSLog(@"🔍 Device ID[%u]: %u", (unsigned int)i, (unsigned int)deviceIDs[i]);
        }
        
        // Step 3 - Check which devices have INPUT capabilities
        NSLog(@"🔍 Step 3 - Checking input capabilities for each device");
        NSMutableArray *inputDevices = [[NSMutableArray alloc] init];
        
        for (UInt32 i = 0; i < deviceCount; i++) {
            AudioDeviceID deviceID = deviceIDs[i];
            NSLog(@"🔍 Checking device ID %u for input capabilities...", (unsigned int)deviceID);
            
            // Check input stream configuration
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeInput;  // INPUT scope
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status != noErr) {
                NSLog(@"⚠️  Device %u: Can't get input stream config size: %d", (unsigned int)deviceID, (int)status);
                continue;
            }
            
            AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            
            UInt32 inputChannels = 0;
            if (status == noErr) {
                for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                    inputChannels += bufferList->mBuffers[j].mNumberChannels;
                }
                NSLog(@"✅ Device %u has %u input channels", (unsigned int)deviceID, (unsigned int)inputChannels);
            } else {
                NSLog(@"❌ Device %u: Failed to get input stream data: %d", (unsigned int)deviceID, (int)status);
            }
            free(bufferList);
            
            // Only include devices with input channels
            if (inputChannels > 0) {
                NSLog(@"🎤 Device %u is an INPUT device with %u channels", (unsigned int)deviceID, (unsigned int)inputChannels);
                
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
                            NSLog(@"📊 Device %u sample rate range: %.0f - %.0f Hz", (unsigned int)deviceID, minRate, maxRate);
                            
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
                    NSLog(@"⚠️  Device %u: No sample rate info, assuming 44100/48000", (unsigned int)deviceID);
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
                        NSLog(@"📊 Device %u bit depths: %@", (unsigned int)deviceID, bitDepths);
                    }
                    free(formats);
                } else {
                    NSLog(@"⚠️  Device %u: No format info, assuming 16/24 bit", (unsigned int)deviceID);
                    [bitDepths addObjectsFromArray:@[@16, @24]];
                }
                
                [inputDevices addObject:@{
                    @"deviceId": @(deviceID), 
                    @"channels": @(inputChannels),
                    @"sampleRates": sampleRates,
                    @"bitDepths": bitDepths
                }];
            } else {
                NSLog(@"🔇 Device %u has no input channels - skipping", (unsigned int)deviceID);
            }
        }
        
        // Step 4 - Get names for ALL input devices
        NSLog(@"🔍 Step 4 - Getting device names for all %lu input devices", (unsigned long)[inputDevices count]);
        NSMutableArray *jsonDevices = [[NSMutableArray alloc] init];
        
        for (NSDictionary *inputDevice in inputDevices) {
            AudioDeviceID deviceID = [inputDevice[@"deviceId"] unsignedIntValue];
            UInt32 channels = [inputDevice[@"channels"] unsignedIntValue];
            NSArray *sampleRates = inputDevice[@"sampleRates"];
            NSArray *bitDepths = inputDevice[@"bitDepths"];
            
            NSLog(@"🔍 Getting name for input device ID: %u", (unsigned int)deviceID);
            
            propertyAddress.mSelector = kAudioDevicePropertyDeviceName;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            dataSize = 256;
            char deviceName[256];
            
            NSString *realDeviceName = @"Unknown Device";
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, deviceName);
            
            if (status == noErr) {
                realDeviceName = [NSString stringWithUTF8String:deviceName];
                NSLog(@"✅ Device %u name: '%s'", (unsigned int)deviceID, deviceName);
            } else {
                NSLog(@"❌ Failed to get name for device %u: %d", (unsigned int)deviceID, (int)status);
            }
            
            // Add device to JSON array
            NSDictionary *deviceJson = @{
                @"name": realDeviceName,
                @"uid": [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID],
                @"deviceId": @(deviceID),
                @"channelCount": @(channels),
                @"supportedSampleRates": sampleRates,
                @"supportedBitDepths": bitDepths,
                @"isDefault": @NO
            };
            [jsonDevices addObject:deviceJson];
        }
        
        NSLog(@"🔍 Summary: Processed %lu input devices out of %u total devices", (unsigned long)[jsonDevices count], (unsigned int)deviceCount);
        
        // Clean up
        free(deviceIDs);
        
        // Return ALL input devices as JSON array
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        NSLog(@"🔍 Returning ALL input devices JSON: %@", result);
        return strdup([result UTF8String]);
    }
}

char* getAudioOutputDevices(void) {
    @autoreleasepool {
        NSLog(@"🔍 getAudioOutputDevices called");
        
        // Get device count - step 1
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        
        NSLog(@"🔍 OUTPUT Step 1 - AudioObjectGetPropertyDataSize result: %d, dataSize: %u", (int)status, (unsigned int)dataSize);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get device count for outputs: %d", (int)status);
            return strdup("[]");
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        NSLog(@"✅ OUTPUT Step 1 complete - Found %u audio devices", (unsigned int)deviceCount);
        
        // Get actual device IDs - step 2
        NSLog(@"🔍 OUTPUT Step 2 - Allocating memory for %u device IDs", (unsigned int)deviceCount);
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            NSLog(@"❌ Failed to allocate memory for device IDs");
            return strdup("[]");
        }
        
        NSLog(@"🔍 OUTPUT Step 2 - Getting actual device IDs");
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        
        if (status != noErr) {
            NSLog(@"❌ Failed to get device IDs for outputs: %d", (int)status);
            free(deviceIDs);
            return strdup("[]");
        }
        
        NSLog(@"✅ OUTPUT Step 2 complete - Got device IDs successfully");
        
        // Step 3 - Check which devices have OUTPUT capabilities
        NSLog(@"🔍 OUTPUT Step 3 - Checking output capabilities for each device");
        NSMutableArray *outputDevices = [[NSMutableArray alloc] init];
        
        for (UInt32 i = 0; i < deviceCount; i++) {
            AudioDeviceID deviceID = deviceIDs[i];
            NSLog(@"🔍 Checking device ID %u for output capabilities...", (unsigned int)deviceID);
            
            // Check output stream configuration
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeOutput;  // OUTPUT scope
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status != noErr) {
                NSLog(@"⚠️  Device %u: Can't get output stream config size: %d", (unsigned int)deviceID, (int)status);
                continue;
            }
            
            AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
            
            UInt32 outputChannels = 0;
            if (status == noErr) {
                for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                    outputChannels += bufferList->mBuffers[j].mNumberChannels;
                }
                NSLog(@"✅ Device %u has %u output channels", (unsigned int)deviceID, (unsigned int)outputChannels);
            } else {
                NSLog(@"❌ Device %u: Failed to get output stream data: %d", (unsigned int)deviceID, (int)status);
            }
            free(bufferList);
            
            // Only include devices with output channels
            if (outputChannels > 0) {
                NSLog(@"🔊 Device %u is an OUTPUT device with %u channels", (unsigned int)deviceID, (unsigned int)outputChannels);
                
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
                            NSLog(@"📊 OUTPUT Device %u sample rate range: %.0f - %.0f Hz", (unsigned int)deviceID, minRate, maxRate);
                            
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
                    NSLog(@"⚠️  OUTPUT Device %u: No sample rate info, assuming 44100/48000", (unsigned int)deviceID);
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
                        NSLog(@"📊 OUTPUT Device %u bit depths: %@", (unsigned int)deviceID, bitDepths);
                    }
                    free(formats);
                } else {
                    NSLog(@"⚠️  OUTPUT Device %u: No format info, assuming 16/24 bit", (unsigned int)deviceID);
                    [bitDepths addObjectsFromArray:@[@16, @24]];
                }
                
                [outputDevices addObject:@{
                    @"deviceId": @(deviceID), 
                    @"channels": @(outputChannels),
                    @"sampleRates": sampleRates,
                    @"bitDepths": bitDepths
                }];
            } else {
                NSLog(@"🔇 Device %u has no output channels - skipping", (unsigned int)deviceID);
            }
        }
        
        // Step 4 - Get names for ALL output devices
        NSLog(@"🔍 OUTPUT Step 4 - Getting device names for all %lu output devices", (unsigned long)[outputDevices count]);
        NSMutableArray *jsonDevices = [[NSMutableArray alloc] init];
        
        for (NSDictionary *outputDevice in outputDevices) {
            AudioDeviceID deviceID = [outputDevice[@"deviceId"] unsignedIntValue];
            UInt32 channels = [outputDevice[@"channels"] unsignedIntValue];
            NSArray *sampleRates = outputDevice[@"sampleRates"];
            NSArray *bitDepths = outputDevice[@"bitDepths"];
            
            NSLog(@"🔍 Getting name for output device ID: %u", (unsigned int)deviceID);
            
            propertyAddress.mSelector = kAudioDevicePropertyDeviceName;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            dataSize = 256;
            char deviceName[256];
            
            NSString *realDeviceName = @"Unknown Device";
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, deviceName);
            
            if (status == noErr) {
                realDeviceName = [NSString stringWithUTF8String:deviceName];
                NSLog(@"✅ OUTPUT Device %u name: '%s'", (unsigned int)deviceID, deviceName);
            } else {
                NSLog(@"❌ Failed to get name for output device %u: %d", (unsigned int)deviceID, (int)status);
            }
            
            // Add device to JSON array
            NSDictionary *deviceJson = @{
                @"name": realDeviceName,
                @"uid": [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID],
                @"deviceId": @(deviceID),
                @"channelCount": @(channels),
                @"supportedSampleRates": sampleRates,
                @"supportedBitDepths": bitDepths,
                @"isDefault": @NO
            };
            [jsonDevices addObject:deviceJson];
        }
        
        NSLog(@"🔍 OUTPUT Summary: Processed %lu output devices out of %u total devices", (unsigned long)[jsonDevices count], (unsigned int)deviceCount);
        
        // Clean up
        free(deviceIDs);
        
        // Return ALL output devices as JSON array
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ OUTPUT JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        NSLog(@"🔍 Returning ALL output devices JSON: %@", result);
        return strdup([result UTF8String]);
    }
}

char* getDefaultAudioDevices(void) {
    @autoreleasepool {
        NSLog(@"🔍 getDefaultAudioDevices called");
        
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
        } else {
            NSLog(@"✅ Default output device ID: %u", (unsigned int)defaultOutputDeviceID);
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
        } else {
            NSLog(@"✅ Default input device ID: %u", (unsigned int)defaultInputDeviceID);
        }
        
        // Create JSON response with actual device IDs
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
        NSLog(@"🔍 Returning default devices: %@", jsonString);
        return strdup([jsonString UTF8String]);
    }
}

char* getMIDIInputDevices(void) {
    @autoreleasepool {
        NSLog(@"🔍 getMIDIInputDevices called");
        
        // Step 1 - Get MIDI device count
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        NSLog(@"🔍 MIDI INPUT Step 1 - Found %lu MIDI devices", (unsigned long)deviceCount);
        
        if (deviceCount == 0) {
            NSLog(@"🔇 No MIDI devices found");
            return strdup("[]");
        }
        
        // Step 2 - Enumerate MIDI devices and find input endpoints
        NSLog(@"🔍 MIDI INPUT Step 2 - Checking devices for input capabilities");
        NSMutableArray *jsonDevices = [[NSMutableArray alloc] init];
        
        for (ItemCount i = 0; i < deviceCount; i++) {
            MIDIDeviceRef device = MIDIGetDevice(i);
            if (device == 0) {
                NSLog(@"⚠️  MIDI device %lu is invalid", (unsigned long)i);
                continue;
            }
            
            NSLog(@"🔍 Checking MIDI device %lu...", (unsigned long)i);
            
            // Get device name
            CFStringRef deviceName;
            OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyName, &deviceName);
            if (status != noErr) {
                NSLog(@"❌ MIDI device %lu: Can't get name: %d", (unsigned long)i, (int)status);
                continue;
            }
            
            NSString *deviceNameString = (__bridge NSString *)deviceName;
            NSLog(@"✅ MIDI device %lu name: '%@'", (unsigned long)i, deviceNameString);
            
            // Get device unique ID
            SInt32 uniqueID;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyUniqueID, &uniqueID);
            if (status != noErr) {
                NSLog(@"❌ MIDI device %lu: Can't get unique ID: %d", (unsigned long)i, (int)status);
                CFRelease(deviceName);
                continue;
            }
            
            // Check if device is online
            SInt32 isOffline;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyOffline, &isOffline);
            BOOL online = (status != noErr) ? YES : !isOffline; // Assume online if property doesn't exist
            NSLog(@"🔍 MIDI device %lu online status: %s", (unsigned long)i, online ? "YES" : "NO");
            
            // Step 3 - Get entities and input endpoints
            ItemCount entityCount = MIDIDeviceGetNumberOfEntities(device);
            NSLog(@"🔍 MIDI device %lu has %lu entities", (unsigned long)i, (unsigned long)entityCount);
            
            for (ItemCount j = 0; j < entityCount; j++) {
                MIDIEntityRef entity = MIDIDeviceGetEntity(device, j);
                if (entity == 0) continue;
                
                // Get input endpoints (sources)
                ItemCount sourceCount = MIDIEntityGetNumberOfSources(entity);
                NSLog(@"🔍 MIDI entity %lu has %lu input sources", (unsigned long)j, (unsigned long)sourceCount);
                
                for (ItemCount k = 0; k < sourceCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetSource(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name (might be different from device name)
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    NSLog(@"🎹 MIDI INPUT endpoint found: '%@' (endpoint ID: %u)", finalName, (unsigned int)endpoint);
                    
                    // Add to results
                    NSDictionary *deviceJson = @{
                        @"name": finalName,
                        @"uid": [NSString stringWithFormat:@"midi_%d", uniqueID],
                        @"endpointId": @(endpoint),
                        @"isOnline": @(online)
                    };
                    [jsonDevices addObject:deviceJson];
                    
                    if (endpointName) CFRelease(endpointName);
                }
            }
            
            CFRelease(deviceName);
        }
        
        NSLog(@"🔍 MIDI INPUT Summary: Found %lu input endpoints", (unsigned long)[jsonDevices count]);
        
        // Return MIDI input devices as JSON array
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ MIDI INPUT JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        NSLog(@"🔍 Returning MIDI INPUT devices JSON: %@", result);
        return strdup([result UTF8String]);
    }
}

char* getMIDIOutputDevices(void) {
    @autoreleasepool {
        NSLog(@"🔍 getMIDIOutputDevices called");
        
        // Step 1 - Get MIDI device count
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        NSLog(@"🔍 MIDI OUTPUT Step 1 - Found %lu MIDI devices", (unsigned long)deviceCount);
        
        if (deviceCount == 0) {
            NSLog(@"🔇 No MIDI devices found");
            return strdup("[]");
        }
        
        // Step 2 - Enumerate MIDI devices and find output endpoints
        NSLog(@"🔍 MIDI OUTPUT Step 2 - Checking devices for output capabilities");
        NSMutableArray *jsonDevices = [[NSMutableArray alloc] init];
        
        for (ItemCount i = 0; i < deviceCount; i++) {
            MIDIDeviceRef device = MIDIGetDevice(i);
            if (device == 0) {
                NSLog(@"⚠️  MIDI device %lu is invalid", (unsigned long)i);
                continue;
            }
            
            NSLog(@"🔍 Checking MIDI device %lu...", (unsigned long)i);
            
            // Get device name
            CFStringRef deviceName;
            OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyName, &deviceName);
            if (status != noErr) {
                NSLog(@"❌ MIDI device %lu: Can't get name: %d", (unsigned long)i, (int)status);
                continue;
            }
            
            NSString *deviceNameString = (__bridge NSString *)deviceName;
            NSLog(@"✅ MIDI device %lu name: '%@'", (unsigned long)i, deviceNameString);
            
            // Get device unique ID
            SInt32 uniqueID;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyUniqueID, &uniqueID);
            if (status != noErr) {
                NSLog(@"❌ MIDI device %lu: Can't get unique ID: %d", (unsigned long)i, (int)status);
                CFRelease(deviceName);
                continue;
            }
            
            // Check if device is online
            SInt32 isOffline;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyOffline, &isOffline);
            BOOL online = (status != noErr) ? YES : !isOffline; // Assume online if property doesn't exist
            NSLog(@"🔍 MIDI device %lu online status: %s", (unsigned long)i, online ? "YES" : "NO");
            
            // Step 3 - Get entities and output endpoints
            ItemCount entityCount = MIDIDeviceGetNumberOfEntities(device);
            NSLog(@"🔍 MIDI device %lu has %lu entities", (unsigned long)i, (unsigned long)entityCount);
            
            for (ItemCount j = 0; j < entityCount; j++) {
                MIDIEntityRef entity = MIDIDeviceGetEntity(device, j);
                if (entity == 0) continue;
                
                // Get output endpoints (destinations)
                ItemCount destCount = MIDIEntityGetNumberOfDestinations(entity);
                NSLog(@"🔍 MIDI entity %lu has %lu output destinations", (unsigned long)j, (unsigned long)destCount);
                
                for (ItemCount k = 0; k < destCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetDestination(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name (might be different from device name)
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    NSLog(@"🎹 MIDI OUTPUT endpoint found: '%@' (endpoint ID: %u)", finalName, (unsigned int)endpoint);
                    
                    // Add to results
                    NSDictionary *deviceJson = @{
                        @"name": finalName,
                        @"uid": [NSString stringWithFormat:@"midi_%d", uniqueID],
                        @"endpointId": @(endpoint),
                        @"isOnline": @(online)
                    };
                    [jsonDevices addObject:deviceJson];
                    
                    if (endpointName) CFRelease(endpointName);
                }
            }
            
            CFRelease(deviceName);
        }
        
        NSLog(@"🔍 MIDI OUTPUT Summary: Found %lu output endpoints", (unsigned long)[jsonDevices count]);
        
        // Return MIDI output devices as JSON array
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonDevices options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSLog(@"❌ MIDI OUTPUT JSON serialization failed");
            return strdup("[]");
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        NSLog(@"🔍 Returning MIDI OUTPUT devices JSON: %@", result);
        return strdup([result UTF8String]);
    }
}

int getAudioDeviceCount(int isInput) {
    NSLog(@"🔍 getAudioDeviceCount called with isInput: %d", isInput);
    return isInput ? 0 : 1;
}

int getMIDIDeviceCount(int isInput) {
    NSLog(@"🔍 getMIDIDeviceCount called with isInput: %d", isInput);
    return 0;
}
