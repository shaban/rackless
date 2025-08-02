//
//  main.m
//  Standalone Audio Host - Pure Objective-C Command Line Tool
//

#import <Foundation/Foundation.h>
#import <AudioToolbox/AudioToolbox.h>
#import <CoreAudio/CoreAudio.h>
#import <AVFoundation/AVFoundation.h>
#import <CoreMIDI/CoreMIDI.h>
#import <AudioUnit/AudioUnit.h>

// Device Enumeration Functions
NSString* enumerateAudioDevices(BOOL isInput) {
    NSMutableArray* devices = [NSMutableArray array];
    
    AudioObjectPropertyAddress address = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    
    UInt32 dataSize = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &address, 0, NULL, &dataSize);
    if (status != noErr) return @"[]";
    
    UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
    AudioDeviceID* deviceIDs = (AudioDeviceID*)malloc(dataSize);
    
    status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &address, 0, NULL, &dataSize, deviceIDs);
    if (status != noErr) {
        free(deviceIDs);
        return @"[]";
    }
    
    for (UInt32 i = 0; i < deviceCount; i++) {
        AudioDeviceID deviceID = deviceIDs[i];
        
        // Check if device has input/output streams
        address.mSelector = isInput ? kAudioDevicePropertyStreamConfiguration : kAudioDevicePropertyStreamConfiguration;
        address.mScope = isInput ? kAudioDevicePropertyScopeInput : kAudioDevicePropertyScopeOutput;
        
        status = AudioObjectGetPropertyDataSize(deviceID, &address, 0, NULL, &dataSize);
        if (status != noErr) continue;
        
        AudioBufferList* bufferList = (AudioBufferList*)malloc(dataSize);
        status = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &dataSize, bufferList);
        
        if (status == noErr && bufferList->mNumberBuffers > 0) {
            // Get device name
            address.mSelector = kAudioDevicePropertyDeviceNameCFString;
            address.mScope = kAudioObjectPropertyScopeGlobal;
            CFStringRef deviceName = NULL;
            dataSize = sizeof(CFStringRef);
            
            status = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &dataSize, &deviceName);
            if (status == noErr && deviceName) {
                // Get device UID
                address.mSelector = kAudioDevicePropertyDeviceUID;
                CFStringRef deviceUID = NULL;
                dataSize = sizeof(CFStringRef);
                AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &dataSize, &deviceUID);
                
                NSMutableDictionary* device = [NSMutableDictionary dictionary];
                device[@"id"] = @(deviceID);
                device[@"name"] = (__bridge NSString*)deviceName;
                if (deviceUID) {
                    device[@"uid"] = (__bridge NSString*)deviceUID;
                    CFRelease(deviceUID);
                }
                device[@"channels"] = @(bufferList->mNumberBuffers);
                
                [devices addObject:device];
                CFRelease(deviceName);
            }
        }
        free(bufferList);
    }
    
    free(deviceIDs);
    
    NSError* error;
    NSData* jsonData = [NSJSONSerialization dataWithJSONObject:devices options:NSJSONWritingPrettyPrinted error:&error];
    if (error) return @"[]";
    
    return [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
}

NSString* enumerateMIDIDevices(BOOL isInput) {
    NSMutableArray* devices = [NSMutableArray array];
    
    ItemCount deviceCount = isInput ? MIDIGetNumberOfSources() : MIDIGetNumberOfDestinations();
    
    for (ItemCount i = 0; i < deviceCount; i++) {
        MIDIEndpointRef endpoint = isInput ? MIDIGetSource(i) : MIDIGetDestination(i);
        
        CFStringRef name = NULL;
        CFStringRef uid = NULL;
        
        MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &name);
        MIDIObjectGetStringProperty(endpoint, kMIDIPropertyUniqueID, &uid);
        
        NSMutableDictionary* device = [NSMutableDictionary dictionary];
        device[@"id"] = @(endpoint);
        if (name) {
            device[@"name"] = (__bridge NSString*)name;
            CFRelease(name);
        }
        if (uid) {
            device[@"uid"] = (__bridge NSString*)uid;
            CFRelease(uid);
        }
        
        [devices addObject:device];
    }
    
    NSError* error;
    NSData* jsonData = [NSJSONSerialization dataWithJSONObject:devices options:NSJSONWritingPrettyPrinted error:&error];
    if (error) return @"[]";
    
    return [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
}

// Audio Host Configuration
typedef struct {
    double sampleRate;
    int bitDepth;
    int bufferSize;
    BOOL enableTestTone;
    int audioInputDeviceID;    // Audio input device ID
    int audioInputChannel;     // Audio input channel (0-based)
} AudioHostConfig;

// Audio Host Engine
@interface AudioHostEngine : NSObject {
@public
    // Core Audio components
    AudioUnit outputUnit;
    AudioUnit inputUnit;
    
    // Plugin management - using AudioUnit for compatibility with Core Audio approach
    AudioUnit pluginUnit;
    BOOL pluginLoaded;
    NSString* loadedPluginName;
    
    // Plugin input data (for passing guitar input to plugin)
    AudioBufferList* pluginInputData;
    UInt32 pluginInputFrames;
    
    // Configuration
    double sampleRate;
    int bitDepth;
    int bufferSize;
    BOOL enableTestTone;
    int audioInputDeviceID;
    int audioInputChannel;
    
    // State
    BOOL isRunning;
    
    // Test tone generator
    double testTonePhase;
    double testToneFrequency;
    
    // Audio input buffer
    AudioBufferList* inputBufferList;
}

- (instancetype)initWithConfig:(AudioHostConfig)config;
- (BOOL)start;
- (BOOL)stop;
- (BOOL)isRunning;
- (void)setTestToneFrequency:(double)frequency;
- (void)setTestToneEnabled:(BOOL)enabled;
- (BOOL)loadPlugin:(NSString*)componentID;
- (BOOL)unloadPlugin;

@end

// Plugin input callback - provides guitar input to the plugin
static OSStatus PluginInputCallback(void* inRefCon,
                                   AudioUnitRenderActionFlags* ioActionFlags,
                                   const AudioTimeStamp* inTimeStamp,
                                   UInt32 inBusNumber,
                                   UInt32 inNumberFrames,
                                   AudioBufferList* ioData) {
    
    AudioHostEngine* engine = (__bridge AudioHostEngine*)inRefCon;
    
    // Safety checks
    if (!engine || !ioData || !engine->pluginInputData) {
        return kAudioUnitErr_NoConnection;
    }
    
    if (inNumberFrames != engine->pluginInputFrames) {
        return kAudioUnitErr_InvalidParameter;
    }
    
    // Copy our prepared input data to the plugin
    for (UInt32 buffer = 0; buffer < ioData->mNumberBuffers && buffer < engine->pluginInputData->mNumberBuffers; buffer++) {
        memcpy(ioData->mBuffers[buffer].mData,
               engine->pluginInputData->mBuffers[buffer].mData,
               ioData->mBuffers[buffer].mDataByteSize);
    }
    
    return noErr;
}

// Audio render callback - runs on Core Audio's real-time thread
static OSStatus AudioRenderCallback(void* inRefCon,
                                   AudioUnitRenderActionFlags* ioActionFlags,
                                   const AudioTimeStamp* inTimeStamp,
                                   UInt32 inBusNumber,
                                   UInt32 inNumberFrames,
                                   AudioBufferList* ioData) {
    
    AudioHostEngine* engine = (__bridge AudioHostEngine*)inRefCon;
    
    // Safety check
    if (!engine || !ioData) {
        return noErr;
    }
    
    // Get configuration
    double sampleRate = engine->sampleRate;
    double* testTonePhase = &(engine->testTonePhase);
    double testToneFrequency = engine->testToneFrequency;
    BOOL enableTestTone = engine->enableTestTone;
    
    if (enableTestTone) {
        // Generate test tone
        double phaseIncrement = 2.0 * M_PI * testToneFrequency / sampleRate;
        
        // Handle interleaved stereo format (most common)
        if (ioData->mNumberBuffers == 1) {
            Float32* buffer = (Float32*)ioData->mBuffers[0].mData;
            
            for (UInt32 frame = 0; frame < inNumberFrames; frame++) {
                Float32 sample = sin(*testTonePhase) * 0.1; // Low volume
                buffer[frame * 2] = sample;     // Left channel
                buffer[frame * 2 + 1] = sample; // Right channel
                
                *testTonePhase += phaseIncrement;
                if (*testTonePhase >= 2.0 * M_PI) {
                    *testTonePhase -= 2.0 * M_PI;
                }
            }
        }
    } else if (engine->audioInputDeviceID != -1) {
        // Get input from guitar using the same HAL unit
        AudioBufferList inputBufferList = {0};
        inputBufferList.mNumberBuffers = 1;
        inputBufferList.mBuffers[0].mNumberChannels = 2;
        inputBufferList.mBuffers[0].mDataByteSize = inNumberFrames * 2 * sizeof(Float32);
        
        // Allocate temporary buffer for input
        Float32 inputData[inNumberFrames * 2];
        inputBufferList.mBuffers[0].mData = inputData;
        
        // Get input from the HAL unit (input bus = 1)
        // NOTE: We need to use kAudioUnitScope_Output for the input bus
        OSStatus status = AudioUnitRender(engine->outputUnit,
                                        ioActionFlags,
                                        inTimeStamp,
                                        1, // Input bus
                                        inNumberFrames,
                                        &inputBufferList);
        
        if (status == noErr && ioData->mNumberBuffers == 1) {
            Float32* outputBuffer = (Float32*)ioData->mBuffers[0].mData;
            Float32* inputBuffer = (Float32*)inputBufferList.mBuffers[0].mData;
            
            // Apply gain and copy guitar input to output
            // Guitar signals are typically much quieter than line level, so we boost them significantly
            float guitarGain = 1.0f; // Baseline test - no gain applied (was 20.0f)
            
            static int debugCounter = 0;
            float maxInputLevel = 0.0f;
            
            // If plugin is loaded, process through plugin first
            if (engine->pluginLoaded && engine->pluginUnit && engine->pluginInputData) {
                // Prepare guitar input for plugin in our input buffer
                Float32* pluginInputBuffer = (Float32*)engine->pluginInputData->mBuffers[0].mData;
                engine->pluginInputFrames = inNumberFrames;
                
                for (UInt32 frame = 0; frame < inNumberFrames; frame++) {
                    Float32 rawSample = inputBuffer[frame * 2 + engine->audioInputChannel];
                    
                    // Track max input level for debugging
                    float absLevel = fabsf(rawSample);
                    if (absLevel > maxInputLevel) {
                        maxInputLevel = absLevel;
                    }
                    
                    // Send mono guitar to both plugin input channels
                    pluginInputBuffer[frame * 2] = rawSample;     // Left channel
                    pluginInputBuffer[frame * 2 + 1] = rawSample; // Right channel
                }
                
                // Process through plugin
                AudioUnitRenderActionFlags renderFlags = 0;
                OSStatus pluginStatus = AudioUnitRender(engine->pluginUnit,
                                                       &renderFlags,
                                                       inTimeStamp,
                                                       0, // Output bus
                                                       inNumberFrames,
                                                       ioData);
                
                if (pluginStatus == noErr) {
                    // Plugin processed successfully - apply gain to processed output
                    for (UInt32 frame = 0; frame < inNumberFrames; frame++) {
                        Float32 processedLeft = outputBuffer[frame * 2];
                        Float32 processedRight = outputBuffer[frame * 2 + 1];
                        
                        // Apply gain to processed signal
                        outputBuffer[frame * 2] = processedLeft * guitarGain;
                        outputBuffer[frame * 2 + 1] = processedRight * guitarGain;
                    }
                } else {
                    // Plugin failed - fall back to direct processing
                    NSLog(@"⚠️ Plugin processing failed: %d, falling back to direct", (int)pluginStatus);
                    goto direct_processing;
                }
            } else {
            direct_processing:
                // Direct processing without plugin
                for (UInt32 frame = 0; frame < inNumberFrames; frame++) {
                    // Get guitar input from the specified channel
                    Float32 rawSample = inputBuffer[frame * 2 + engine->audioInputChannel];
                    Float32 guitarSample = rawSample * guitarGain;
                    
                    // Track max input level for debugging
                    float absLevel = fabsf(rawSample);
                    if (absLevel > maxInputLevel) {
                        maxInputLevel = absLevel;
                    }
                    
                    // Send to both output channels (mono guitar to stereo output)
                    outputBuffer[frame * 2] = guitarSample;     // Left channel
                    outputBuffer[frame * 2 + 1] = guitarSample; // Right channel
                }
            }
            
            // Debug input levels every few seconds
            debugCounter++;
            if (debugCounter >= 2000) { // Print every ~2 seconds at 44.1kHz with 256 buffer
                if (maxInputLevel > 0.0001f) {
                    NSString* pluginStatus = engine->pluginLoaded ? 
                        [NSString stringWithFormat:@"→ %@", engine->loadedPluginName] : @"(no plugin)";
                    NSLog(@"🎸 Guitar input detected - Level: %.6f (gained: %.6f) %@", 
                          maxInputLevel, maxInputLevel * guitarGain, pluginStatus);
                } else {
                    NSLog(@"🔇 No guitar input detected (max level: %.6f)", maxInputLevel);
                }
                debugCounter = 0;
            }
        } else {
            if (status != noErr) {
                // Only log occasionally to avoid spam
                static int errorCounter = 0;
                errorCounter++;
                if (errorCounter >= 1000) { // Log every few seconds
                    NSLog(@"⚠️ Cannot get input from HAL unit (error: %d). Generating silence.", (int)status);
                    errorCounter = 0;
                }
            }
            
            // If input failed, generate silence
            for (UInt32 buffer = 0; buffer < ioData->mNumberBuffers; buffer++) {
                memset(ioData->mBuffers[buffer].mData, 0, ioData->mBuffers[buffer].mDataByteSize);
            }
        }
    } else {
        // Generate silence
        for (UInt32 buffer = 0; buffer < ioData->mNumberBuffers; buffer++) {
            memset(ioData->mBuffers[buffer].mData, 0, ioData->mBuffers[buffer].mDataByteSize);
        }
    }
    
    return noErr;
}

@implementation AudioHostEngine

- (instancetype)initWithConfig:(AudioHostConfig)config {
    self = [super init];
    if (self) {
        sampleRate = config.sampleRate;
        bitDepth = config.bitDepth;
        bufferSize = config.bufferSize;
        enableTestTone = config.enableTestTone;
        audioInputDeviceID = config.audioInputDeviceID;
        audioInputChannel = config.audioInputChannel;
        isRunning = NO;
        
        // Plugin management setup
        pluginUnit = NULL;
        pluginLoaded = NO;
        loadedPluginName = nil;
        pluginInputData = NULL;
        pluginInputFrames = 0;
        
        // Test tone setup
        testTonePhase = 0.0;
        testToneFrequency = 440.0; // A4 note
        
        NSLog(@"🎵 AudioHostEngine initialized:");
        NSLog(@"   Sample Rate: %.0f Hz", sampleRate);
        NSLog(@"   Bit Depth: %d", bitDepth);
        NSLog(@"   Buffer Size: %d samples", bufferSize);
        NSLog(@"   Test Tone: %@", enableTestTone ? @"ON" : @"OFF");
        if (audioInputDeviceID != -1) {
            NSLog(@"   Audio Input: Device %d, Channel %d", audioInputDeviceID, audioInputChannel);
        } else {
            NSLog(@"   Audio Input: None");
        }
    }
    return self;
}

- (BOOL)start {
    if (isRunning) {
        NSLog(@"⚠️  Audio host already running");
        return YES;
    }
    
    OSStatus status;
    
    // Use a single AUHAL unit that can handle both input and output
    AudioComponentDescription halDesc = {0};
    halDesc.componentType = kAudioUnitType_Output;
    halDesc.componentSubType = kAudioUnitSubType_HALOutput;
    halDesc.componentManufacturer = kAudioUnitManufacturer_Apple;
    
    AudioComponent halComp = AudioComponentFindNext(NULL, &halDesc);
    if (!halComp) {
        NSLog(@"❌ Failed to find HAL AudioUnit");
        return NO;
    }
    
    status = AudioComponentInstanceNew(halComp, &outputUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to create HAL AudioUnit: %d", (int)status);
        return NO;
    }
    
    // Enable input on the HAL unit if we have an input device
    if (audioInputDeviceID != -1) {
        UInt32 enableInput = 1;
        status = AudioUnitSetProperty(outputUnit,
                                     kAudioOutputUnitProperty_EnableIO,
                                     kAudioUnitScope_Input,
                                     1, // Input bus
                                     &enableInput,
                                     sizeof(enableInput));
        if (status != noErr) {
            NSLog(@"❌ Failed to enable input on HAL unit: %d", (int)status);
            return NO;
        }
        
        // Report input device sample rate for server validation
        AudioObjectPropertyAddress address = {
            kAudioDevicePropertyNominalSampleRate,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        Float64 deviceSampleRate = 0;
        UInt32 propSize = sizeof(deviceSampleRate);
        OSStatus propStatus = AudioObjectGetPropertyData(audioInputDeviceID, &address, 0, NULL, &propSize, &deviceSampleRate);
        if (propStatus == noErr) {
            NSLog(@"🔍 Input device %d sample rate: %.0f Hz (engine: %.0f Hz)", audioInputDeviceID, deviceSampleRate, sampleRate);
            if (deviceSampleRate != sampleRate) {
                NSLog(@"❌ SAMPLE_RATE_MISMATCH: Input device %d is at %.0f Hz but engine expects %.0f Hz", 
                      audioInputDeviceID, deviceSampleRate, sampleRate);
                NSLog(@"💡 Server should ensure device sample rate matches before sending start command");
                return NO;
            }
        } else {
            NSLog(@"❌ SAMPLE_RATE_CHECK_FAILED: Could not verify input device sample rate: %d", (int)propStatus);
            return NO;
        }
        
        // Set input device
        status = AudioUnitSetProperty(outputUnit,
                                     kAudioOutputUnitProperty_CurrentDevice,
                                     kAudioUnitScope_Global,
                                     1, // Input side
                                     &audioInputDeviceID,
                                     sizeof(audioInputDeviceID));
        if (status != noErr) {
            NSLog(@"❌ Failed to set input device: %d", (int)status);
            return NO;
        }
        
        NSLog(@"✅ HAL AudioUnit configured for input device %d", audioInputDeviceID);
    }
    
    // Enable output on the HAL unit (should be enabled by default, but let's be explicit)
    UInt32 enableOutput = 1;
    status = AudioUnitSetProperty(outputUnit,
                                 kAudioOutputUnitProperty_EnableIO,
                                 kAudioUnitScope_Output,
                                 0, // Output bus
                                 &enableOutput,
                                 sizeof(enableOutput));
    if (status != noErr) {
        NSLog(@"❌ Failed to enable output on HAL unit: %d", (int)status);
        return NO;
    }
    
    // Report output device sample rate for server validation  
    AudioDeviceID outputDeviceID = 0;
    UInt32 propSize = sizeof(outputDeviceID);
    status = AudioUnitGetProperty(outputUnit,
                                 kAudioOutputUnitProperty_CurrentDevice,
                                 kAudioUnitScope_Global,
                                 0, // Output side
                                 &outputDeviceID,
                                 &propSize);
    if (status == noErr && outputDeviceID != 0) {
        AudioObjectPropertyAddress address = {
            kAudioDevicePropertyNominalSampleRate,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        Float64 deviceSampleRate = 0;
        propSize = sizeof(deviceSampleRate);
        OSStatus propStatus = AudioObjectGetPropertyData(outputDeviceID, &address, 0, NULL, &propSize, &deviceSampleRate);
        if (propStatus == noErr) {
            NSLog(@"🔍 Output device %d sample rate: %.0f Hz (engine: %.0f Hz)", outputDeviceID, deviceSampleRate, sampleRate);
            if (deviceSampleRate != sampleRate) {
                NSLog(@"❌ SAMPLE_RATE_MISMATCH: Output device %d is at %.0f Hz but engine expects %.0f Hz", 
                      outputDeviceID, deviceSampleRate, sampleRate);
                NSLog(@"💡 Server should ensure device sample rate matches before sending start command");
                return NO;
            }
        } else {
            NSLog(@"❌ SAMPLE_RATE_CHECK_FAILED: Could not verify output device sample rate: %d", (int)propStatus);
            return NO;
        }
    } else {
        NSLog(@"❌ DEVICE_ID_CHECK_FAILED: Could not get output device ID: %d", (int)status);
        return NO;
    }
    
    // Configure audio format for the HAL unit
    AudioStreamBasicDescription format = {0};
    format.mSampleRate = sampleRate;
    format.mFormatID = kAudioFormatLinearPCM;
    format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
    format.mBitsPerChannel = 32; // Use 32-bit float internally
    format.mChannelsPerFrame = 2; // Stereo
    format.mBytesPerFrame = format.mChannelsPerFrame * sizeof(Float32);
    format.mFramesPerPacket = 1;
    format.mFramesPerPacket = 1;
    format.mBytesPerPacket = format.mBytesPerFrame;
    
    // Set format for output (bus 0)
    status = AudioUnitSetProperty(outputUnit,
                                 kAudioUnitProperty_StreamFormat,
                                 kAudioUnitScope_Input,
                                 0, // Output bus
                                 &format,
                                 sizeof(format));
    if (status != noErr) {
        NSLog(@"❌ Failed to set output format: %d", (int)status);
        return NO;
    }
    
    // Set format for input (bus 1) if we have input
    if (audioInputDeviceID != -1) {
        status = AudioUnitSetProperty(outputUnit,
                                     kAudioUnitProperty_StreamFormat,
                                     kAudioUnitScope_Output,
                                     1, // Input bus
                                     &format,
                                     sizeof(format));
        if (status != noErr) {
            NSLog(@"❌ Failed to set input format: %d", (int)status);
            return NO;
        }
    }
    
    // Set render callback
    AURenderCallbackStruct callback = {0};
    callback.inputProc = AudioRenderCallback;
    callback.inputProcRefCon = (__bridge void*)self;
    
    status = AudioUnitSetProperty(outputUnit,
                                 kAudioUnitProperty_SetRenderCallback,
                                 kAudioUnitScope_Input,
                                 0,
                                 &callback,
                                 sizeof(callback));
    if (status != noErr) {
        NSLog(@"❌ Failed to set render callback: %d", (int)status);
        return NO;
    }
    
    // Initialize and start
    status = AudioUnitInitialize(outputUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to initialize AudioUnit: %d", (int)status);
        return NO;
    }
    
    status = AudioOutputUnitStart(outputUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to start AudioUnit: %d", (int)status);
        return NO;
    }
    
    isRunning = YES;
    NSLog(@"✅ Audio host started successfully!");
    
    return YES;
}

- (BOOL)stop {
    if (!isRunning) {
        return YES;
    }
    
    // Clean up plugin first
    if (pluginLoaded) {
        [self unloadPlugin];
    }
    
    if (outputUnit) {
        AudioOutputUnitStop(outputUnit);
        AudioUnitUninitialize(outputUnit);
        AudioComponentInstanceDispose(outputUnit);
        outputUnit = NULL;
    }
    
    isRunning = NO;
    NSLog(@"🔇 Audio host stopped");
    return YES;
}

- (BOOL)isRunning {
    return isRunning;
}

- (void)setTestToneFrequency:(double)frequency {
    testToneFrequency = frequency;
    NSLog(@"🎵 Test tone frequency set to %.1f Hz", frequency);
}

- (void)setTestToneEnabled:(BOOL)enabled {
    enableTestTone = enabled;
    NSLog(@"🎵 Test tone %@", enabled ? @"enabled" : @"disabled");
}

// Helper function to convert string identifiers to FourCharCodes
OSType stringToFourCC(NSString* str) {
    if (str.length != 4) return 0;
    const char* chars = [str UTF8String];
    return (chars[0] << 24) | (chars[1] << 16) | (chars[2] << 8) | chars[3];
}

// Helper function to convert FourCharCode back to string
NSString* fourCCToString(OSType code) {
    char str[5] = {0};
    str[0] = (code >> 24) & 0xFF;
    str[1] = (code >> 16) & 0xFF;
    str[2] = (code >> 8) & 0xFF;
    str[3] = code & 0xFF;
    return [NSString stringWithUTF8String:str];
}

- (BOOL)loadPlugin:(NSString*)componentID {
    if (pluginLoaded) {
        NSLog(@"❌ Plugin already loaded. Unload first.");
        return NO;
    }
    
    NSArray* parts = [componentID componentsSeparatedByString:@":"];
    if (parts.count != 3) {
        NSLog(@"❌ Invalid component ID format. Expected 'type:subtype:manufacturer'");
        return NO;
    }
    
    // Convert string identifiers to FourCharCodes
    OSType type = stringToFourCC(parts[0]);
    OSType subtype = stringToFourCC(parts[1]);
    OSType manufacturer = stringToFourCC(parts[2]);
    
    if (type == 0 || subtype == 0 || manufacturer == 0) {
        NSLog(@"❌ Invalid component identifiers");
        return NO;
    }
    
    NSLog(@"🔄 Loading plugin: %@", componentID);
    
    AudioComponentDescription desc = {
        .componentType = type,
        .componentSubType = subtype,
        .componentManufacturer = manufacturer,
        .componentFlags = 0,
        .componentFlagsMask = 0
    };
    
    // Find the component
    AudioComponent component = AudioComponentFindNext(NULL, &desc);
    if (!component) {
        NSLog(@"❌ Plugin component not found");
        return NO;
    }
    
    // Create AudioUnit instance
    OSStatus status = AudioComponentInstanceNew(component, &pluginUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to create plugin instance: %d", (int)status);
        return NO;
    }
    
    // Configure audio format for plugin
    AudioStreamBasicDescription format = {0};
    format.mSampleRate = sampleRate;
    format.mFormatID = kAudioFormatLinearPCM;
    format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
    format.mBitsPerChannel = 32;
    format.mChannelsPerFrame = 2; // Stereo
    format.mBytesPerFrame = format.mChannelsPerFrame * sizeof(Float32);
    format.mFramesPerPacket = 1;
    format.mBytesPerPacket = format.mBytesPerFrame;
    
    // Set input format
    status = AudioUnitSetProperty(pluginUnit,
                                 kAudioUnitProperty_StreamFormat,
                                 kAudioUnitScope_Input,
                                 0,
                                 &format,
                                 sizeof(format));
    if (status != noErr) {
        NSLog(@"❌ Failed to set plugin input format: %d", (int)status);
        AudioComponentInstanceDispose(pluginUnit);
        pluginUnit = NULL;
        return NO;
    }
    
    // Set output format
    status = AudioUnitSetProperty(pluginUnit,
                                 kAudioUnitProperty_StreamFormat,
                                 kAudioUnitScope_Output,
                                 0,
                                 &format,
                                 sizeof(format));
    if (status != noErr) {
        NSLog(@"❌ Failed to set plugin output format: %d", (int)status);
        AudioComponentInstanceDispose(pluginUnit);
        pluginUnit = NULL;
        return NO;
    }
    
    // Initialize the plugin
    status = AudioUnitInitialize(pluginUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to initialize plugin: %d", (int)status);
        AudioComponentInstanceDispose(pluginUnit);
        pluginUnit = NULL;
        return NO;
    }
    
    // Set up input callback for the plugin
    AURenderCallbackStruct inputCallback = {0};
    inputCallback.inputProc = PluginInputCallback;
    inputCallback.inputProcRefCon = (__bridge void*)self;
    
    status = AudioUnitSetProperty(pluginUnit,
                                 kAudioUnitProperty_SetRenderCallback,
                                 kAudioUnitScope_Input,
                                 0,
                                 &inputCallback,
                                 sizeof(inputCallback));
    if (status != noErr) {
        NSLog(@"❌ Failed to set plugin input callback: %d", (int)status);
        AudioUnitUninitialize(pluginUnit);
        AudioComponentInstanceDispose(pluginUnit);
        pluginUnit = NULL;
        return NO;
    }
    
    // Allocate plugin input buffer (use configured buffer size + safety margin)
    UInt32 maxFrames = self->bufferSize * 2; // 2x buffer size for safety
    if (maxFrames < 64) maxFrames = 64; // Minimum safety buffer
    if (maxFrames > 2048) maxFrames = 2048; // Maximum reasonable buffer
    
    NSLog(@"🔧 Allocating plugin buffer: %u frames (engine buffer: %d)", maxFrames, self->bufferSize);
    pluginInputData = (AudioBufferList*)malloc(sizeof(AudioBufferList) + sizeof(AudioBuffer));
    pluginInputData->mNumberBuffers = 1;
    pluginInputData->mBuffers[0].mNumberChannels = 2;
    pluginInputData->mBuffers[0].mDataByteSize = maxFrames * 2 * sizeof(Float32);
    pluginInputData->mBuffers[0].mData = malloc(pluginInputData->mBuffers[0].mDataByteSize);
    
    pluginLoaded = YES;
    loadedPluginName = [componentID copy];
    
    NSLog(@"✅ Plugin loaded successfully: %@:%@:%@", 
          fourCCToString(type), fourCCToString(subtype), fourCCToString(manufacturer));
    
    return YES;
}

- (BOOL)unloadPlugin {
    if (!pluginLoaded) {
        NSLog(@"⚠️  No plugin loaded");
        return YES;
    }
    
    if (pluginUnit) {
        AudioUnitUninitialize(pluginUnit);
        AudioComponentInstanceDispose(pluginUnit);
        pluginUnit = NULL;
    }
    
    // Clean up plugin input buffer
    if (pluginInputData) {
        if (pluginInputData->mBuffers[0].mData) {
            free(pluginInputData->mBuffers[0].mData);
        }
        free(pluginInputData);
        pluginInputData = NULL;
    }
    pluginInputFrames = 0;
    
    pluginLoaded = NO;
    loadedPluginName = nil;
    
    NSLog(@"✅ Plugin unloaded");
    return YES;
}

@end

// Command processor
void processCommand(AudioHostEngine* engine, NSString* command) {
    NSArray* parts = [command componentsSeparatedByString:@" "];
    NSString* cmd = [parts firstObject];
    
    if ([cmd isEqualToString:@"start"]) {
        if ([engine start]) {
            printf("OK: started\n");
        } else {
            printf("ERROR: failed to start\n");
        }
    }
    else if ([cmd isEqualToString:@"stop"]) {
        if ([engine stop]) {
            printf("OK: stopped\n");
        } else {
            printf("ERROR: failed to stop\n");
        }
    }
    else if ([cmd isEqualToString:@"status"]) {
        printf("STATUS: running=%s sampleRate=%.0f bufferSize=%d testTone=%s toneFreq=%.1f\n",
               [engine isRunning] ? "true" : "false",
               engine->sampleRate,
               engine->bufferSize,
               engine->enableTestTone ? "true" : "false",
               engine->testToneFrequency);
    }
    else if ([cmd isEqualToString:@"tone"] && parts.count >= 2) {
        NSString* subCmd = parts[1];
        if ([subCmd isEqualToString:@"on"]) {
            [engine setTestToneEnabled:YES];
            printf("OK: tone enabled\n");
        }
        else if ([subCmd isEqualToString:@"off"]) {
            [engine setTestToneEnabled:NO];
            printf("OK: tone disabled\n");
        }
        else if ([subCmd isEqualToString:@"freq"] && parts.count >= 3) {
            double freq = [parts[2] doubleValue];
            if (freq > 0 && freq <= 20000) {
                [engine setTestToneFrequency:freq];
                printf("OK: frequency set to %.1f\n", freq);
            } else {
                printf("ERROR: invalid frequency (0-20000 Hz)\n");
            }
        }
        else {
            printf("ERROR: unknown tone command\n");
        }
    }
    else if ([cmd isEqualToString:@"devices"]) {
        if (parts.count >= 2) {
            NSString* deviceType = parts[1];
            if ([deviceType isEqualToString:@"audio-input"]) {
                NSString* json = enumerateAudioDevices(YES);
                printf("%s\n", [json UTF8String]);  // Clean JSON output
            }
            else if ([deviceType isEqualToString:@"audio-output"]) {
                NSString* json = enumerateAudioDevices(NO);
                printf("%s\n", [json UTF8String]);  // Clean JSON output
            }
            else if ([deviceType isEqualToString:@"midi-input"]) {
                NSString* json = enumerateMIDIDevices(YES);
                printf("%s\n", [json UTF8String]);  // Clean JSON output
            }
            else if ([deviceType isEqualToString:@"midi-output"]) {
                NSString* json = enumerateMIDIDevices(NO);
                printf("%s\n", [json UTF8String]);  // Clean JSON output
            }
            else {
                printf("ERROR: unknown device type (audio-input|audio-output|midi-input|midi-output)\n");
            }
        } else {
            printf("ERROR: device type required\n");
        }
    }
    else if ([cmd isEqualToString:@"load-plugin"] && parts.count >= 2) {
        NSString* componentID = parts[1];
        if ([engine loadPlugin:componentID]) {
            printf("OK: plugin loaded\n");
        } else {
            printf("ERROR: failed to load plugin\n");
        }
    }
    else if ([cmd isEqualToString:@"unload-plugin"]) {
        if ([engine unloadPlugin]) {
            printf("OK: plugin unloaded\n");
        } else {
            printf("ERROR: failed to unload plugin\n");
        }
    }
    else if ([cmd isEqualToString:@"list-plugins"]) {
        if (engine->pluginLoaded) {
            printf("LOADED: %s\n", [engine->loadedPluginName UTF8String]);
        } else {
            printf("LOADED: none\n");
        }
    }
    else if ([cmd isEqualToString:@"quit"] || [cmd isEqualToString:@"exit"]) {
        [engine stop];
        printf("OK: goodbye\n");
        exit(0);
    }
    else if ([cmd isEqualToString:@"help"]) {
        printf("Commands:\n");
        printf("  start              - Start audio processing\n");
        printf("  stop               - Stop audio processing\n");
        printf("  status             - Get current status\n");
        printf("  tone on|off        - Enable/disable test tone\n");
        printf("  tone freq <hz>     - Set test tone frequency\n");
        printf("  load-plugin <id>   - Load plugin (format: type:subtype:manufacturer)\n");
        printf("  unload-plugin      - Unload current plugin\n");
        printf("  list-plugins       - Show loaded plugins\n");
        printf("  devices <type>     - Enumerate devices (audio-input|audio-output|midi-input|midi-output)\n");
        printf("  quit|exit          - Stop and exit\n");
        printf("  help               - Show this help\n");
    }
    else {
        printf("ERROR: unknown command '%s' (try 'help')\n", [cmd UTF8String]);
    }
    
    fflush(stdout);
}

// Main function
int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // Default configuration - sample rate must be provided via command line
        AudioHostConfig config = {
            .sampleRate = 44100.0, // Default fallback - should be overridden
            .bitDepth = 32,
            .bufferSize = 256,
            .enableTestTone = NO,  // Disable test tone by default to hear guitar input
            .audioInputDeviceID = -1,  // No input device by default
            .audioInputChannel = 0     // Default to channel 0 (first channel)
        };
        
        BOOL interactiveMode = YES;
        BOOL commandMode = NO;
        
        // Parse command line arguments
        for (int i = 1; i < argc; i++) {
            if (strcmp(argv[i], "--no-tone") == 0) {
                config.enableTestTone = NO;
            } else if (strcmp(argv[i], "--sample-rate") == 0 && i + 1 < argc) {
                config.sampleRate = atof(argv[++i]);
            } else if (strcmp(argv[i], "--buffer-size") == 0 && i + 1 < argc) {
                config.bufferSize = atoi(argv[++i]);
            } else if (strcmp(argv[i], "--audio-input-device") == 0 && i + 1 < argc) {
                config.audioInputDeviceID = atoi(argv[++i]);
            } else if (strcmp(argv[i], "--audio-input-channel") == 0 && i + 1 < argc) {
                config.audioInputChannel = atoi(argv[++i]);
            } else if (strcmp(argv[i], "--command-mode") == 0) {
                commandMode = YES;
                interactiveMode = NO;
            } else if (strcmp(argv[i], "--help") == 0) {
                printf("Usage: %s [options]\n", argv[0]);
                printf("Options:\n");
                printf("  --no-tone                    Disable test tone\n");
                printf("  --sample-rate <hz>           Set sample rate (REQUIRED)\n");
                printf("  --buffer-size <n>            Set buffer size (default: 256)\n");
                printf("  --audio-input-device <id>    Set audio input device ID\n");
                printf("  --audio-input-channel <n>    Set audio input channel (0-based, default: 0)\n");
                printf("  --command-mode               Run in command mode (stdin/stdout)\n");
                printf("  --help                       Show this help\n");
                return 0;
            }
        }
        
        // Create audio engine
        AudioHostEngine* engine = [[AudioHostEngine alloc] initWithConfig:config];
        
        if (commandMode) {
            // Command mode: read commands from stdin, write responses to stdout
            // Send ready signal to stderr so stdout stays clean for JSON
            fprintf(stderr, "READY\n");
            fflush(stderr);
            
            char buffer[1024];
            while (fgets(buffer, sizeof(buffer), stdin)) {
                // Remove newline
                buffer[strcspn(buffer, "\n")] = 0;
                
                NSString* command = [NSString stringWithUTF8String:buffer];
                processCommand(engine, command);
            }
        } else {
            // Interactive mode: original behavior
            NSLog(@"🎶 Standalone Audio Host");
            NSLog(@"========================");
            
            if (![engine start]) {
                NSLog(@"❌ Failed to start audio host");
                return 1;
            }
            
            if (config.enableTestTone) {
                NSLog(@"🎵 Playing 440Hz test tone...");
            } else {
                NSLog(@"🔇 Generating silence...");
            }
            NSLog(@"Press Ctrl+C to stop");
            
            // Keep the program running
            [[NSRunLoop currentRunLoop] run];
        }
    }
    
    return 0;
}
