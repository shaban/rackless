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

// Get system default sample rate
Float64 getSystemDefaultSampleRate() {
    AudioDeviceID defaultDevice;
    UInt32 size = sizeof(AudioDeviceID);
    AudioObjectPropertyAddress address = {
        kAudioHardwarePropertyDefaultOutputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    
    OSStatus status = AudioObjectGetPropertyData(kAudioObjectSystemObject, 
                                               &address, 0, NULL, &size, &defaultDevice);
    if (status != noErr) {
        NSLog(@"⚠️  Failed to get default output device, using 44100 Hz");
        return 44100.0; // fallback
    }
    
    address.mSelector = kAudioDevicePropertyNominalSampleRate;
    Float64 sampleRate;
    size = sizeof(Float64);
    status = AudioObjectGetPropertyData(defaultDevice, &address, 0, NULL, &size, &sampleRate);
    
    if (status != noErr) {
        NSLog(@"⚠️  Failed to get device sample rate, using 44100 Hz");
        return 44100.0; // fallback
    }
    
    NSLog(@"🎵 Detected system sample rate: %.0f Hz", sampleRate);
    return sampleRate;
}

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
} AudioHostConfig;

// Audio Host Engine
@interface AudioHostEngine : NSObject {
@public
    // Core Audio components
    AudioUnit outputUnit;
    
    // Configuration
    double sampleRate;
    int bitDepth;
    int bufferSize;
    BOOL enableTestTone;
    
    // State
    BOOL isRunning;
    
    // Test tone generator
    double testTonePhase;
    double testToneFrequency;
}

- (instancetype)initWithConfig:(AudioHostConfig)config;
- (BOOL)start;
- (BOOL)stop;
- (BOOL)isRunning;
- (void)setTestToneFrequency:(double)frequency;
- (void)setTestToneEnabled:(BOOL)enabled;

@end

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
        isRunning = NO;
        
        // Test tone setup
        testTonePhase = 0.0;
        testToneFrequency = 440.0; // A4 note
        
        NSLog(@"🎵 AudioHostEngine initialized:");
        NSLog(@"   Sample Rate: %.0f Hz", sampleRate);
        NSLog(@"   Bit Depth: %d", bitDepth);
        NSLog(@"   Buffer Size: %d samples", bufferSize);
        NSLog(@"   Test Tone: %@", enableTestTone ? @"ON" : @"OFF");
    }
    return self;
}

- (BOOL)start {
    if (isRunning) {
        NSLog(@"⚠️  Audio host already running");
        return YES;
    }
    
    OSStatus status;
    
    // Create output AudioUnit
    AudioComponentDescription outputDesc = {0};
    outputDesc.componentType = kAudioUnitType_Output;
    outputDesc.componentSubType = kAudioUnitSubType_DefaultOutput;
    outputDesc.componentManufacturer = kAudioUnitManufacturer_Apple;
    
    AudioComponent outputComp = AudioComponentFindNext(NULL, &outputDesc);
    if (!outputComp) {
        NSLog(@"❌ Failed to find default output AudioUnit");
        return NO;
    }
    
    status = AudioComponentInstanceNew(outputComp, &outputUnit);
    if (status != noErr) {
        NSLog(@"❌ Failed to create output AudioUnit: %d", (int)status);
        return NO;
    }
    
    // Configure audio format
    AudioStreamBasicDescription format = {0};
    format.mSampleRate = sampleRate;
    format.mFormatID = kAudioFormatLinearPCM;
    format.mFormatFlags = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
    format.mBitsPerChannel = 32; // Use 32-bit float internally
    format.mChannelsPerFrame = 2; // Stereo
    format.mBytesPerFrame = format.mChannelsPerFrame * sizeof(Float32);
    format.mFramesPerPacket = 1;
    format.mBytesPerPacket = format.mBytesPerFrame;
    
    status = AudioUnitSetProperty(outputUnit,
                                 kAudioUnitProperty_StreamFormat,
                                 kAudioUnitScope_Input,
                                 0,
                                 &format,
                                 sizeof(format));
    if (status != noErr) {
        NSLog(@"❌ Failed to set output format: %d", (int)status);
        return NO;
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
        // Default configuration
        AudioHostConfig config = {
            .sampleRate = getSystemDefaultSampleRate(),
            .bitDepth = 32,
            .bufferSize = 256,
            .enableTestTone = YES
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
            } else if (strcmp(argv[i], "--command-mode") == 0) {
                commandMode = YES;
                interactiveMode = NO;
            } else if (strcmp(argv[i], "--help") == 0) {
                printf("Usage: %s [options]\n", argv[0]);
                printf("Options:\n");
                printf("  --no-tone           Disable test tone\n");
                printf("  --sample-rate <hz>  Set sample rate (default: 44100)\n");
                printf("  --buffer-size <n>   Set buffer size (default: 256)\n");
                printf("  --command-mode      Run in command mode (stdin/stdout)\n");
                printf("  --help              Show this help\n");
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
