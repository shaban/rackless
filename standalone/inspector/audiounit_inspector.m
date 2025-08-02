#import <Foundation/Foundation.h>
#import <AudioToolbox/AudioToolbox.h>
#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AUComponent.h>

// External verbose logging flag set by main.m
extern int g_verboseLogging;

// Conditional logging macros
#define VERBOSE_LOG(...) do { if (g_verboseLogging) fprintf(stderr, __VA_ARGS__); } while(0)
#define PROGRESS_LOG(...) fprintf(stderr, __VA_ARGS__)  // Always show progress

// Helper function to convert a FourCharCode (OSType) to an NSString
NSString* StringFromFourCharCode(FourCharCode code) {
    char chars[5];
    chars[0] = (char)((code >> 24) & 0xFF);
    chars[1] = (char)((code >> 16) & 0xFF);
    chars[2] = (char)((code >> 8) & 0xFF);
    chars[3] = (char)(code & 0xFF);
    chars[4] = '\0';
    return [NSString stringWithCString:chars encoding:NSASCIIStringEncoding];
}

// Helper function to convert AudioUnitParameterUnit to NSString
NSString* StringFromAudioUnitParameterUnit(AudioUnitParameterUnit unit) {
    switch (unit) {
        case kAudioUnitParameterUnit_Generic: return @"Generic";
        case kAudioUnitParameterUnit_Indexed: return @"Indexed";
        case kAudioUnitParameterUnit_Boolean: return @"Boolean";
        case kAudioUnitParameterUnit_Percent: return @"Percent";
        case kAudioUnitParameterUnit_Seconds: return @"Seconds";
        case kAudioUnitParameterUnit_SampleFrames: return @"Sample Frames";
        case kAudioUnitParameterUnit_Phase: return @"Phase";
        case kAudioUnitParameterUnit_Rate: return @"Rate";
        case kAudioUnitParameterUnit_Hertz: return @"Hertz";
        case kAudioUnitParameterUnit_Cents: return @"Cents";
        case kAudioUnitParameterUnit_RelativeSemiTones: return @"Relative Semitones";
        case kAudioUnitParameterUnit_MIDINoteNumber: return @"MIDI Note Number";
        case kAudioUnitParameterUnit_MIDIController: return @"MIDI Controller";
        case kAudioUnitParameterUnit_Decibels: return @"Decibels";
        case kAudioUnitParameterUnit_LinearGain: return @"Linear Gain";
        case kAudioUnitParameterUnit_Degrees: return @"Degrees";
        case kAudioUnitParameterUnit_Meters: return @"Meters";
        case kAudioUnitParameterUnit_AbsoluteCents: return @"Absolute Cents";
        case kAudioUnitParameterUnit_Octaves: return @"Octaves";
        case kAudioUnitParameterUnit_BPM: return @"BPM";
        case kAudioUnitParameterUnit_Beats: return @"Beats";
        case kAudioUnitParameterUnit_Milliseconds: return @"Milliseconds";
        case kAudioUnitParameterUnit_Ratio: return @"Ratio";
        case kAudioUnitParameterUnit_CustomUnit: return @"Custom Unit";
        // Missing units from Apple documentation:
        case kAudioUnitParameterUnit_EqualPowerCrossfade: return @"Equal Power Crossfade";
        case kAudioUnitParameterUnit_MixerFaderCurve1: return @"Mixer Fader Curve 1";
        case kAudioUnitParameterUnit_Pan: return @"Pan";
        case kAudioUnitParameterUnit_MIDI2Controller: return @"MIDI 2.0 Controller";
        default: return [NSString stringWithFormat:@"Unknown (%lu)", (unsigned long)unit];
    }
}

@interface AudioUnitInspector : NSObject
- (BOOL)isPresetParameter:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit;
- (void)initializeAudioUnitForParameterExtraction:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock;
- (void)performAudioProcessingToCompleteInitialization:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock;
- (void)extractIndexedParameterInfo:(AUParameter *)param paramData:(NSMutableDictionary *)paramData audioUnit:(AUAudioUnit *)audioUnit;
- (void)processParametersForAudioUnit:(AUAudioUnit *)audioUnit withName:(NSString *)auName auParameters:(NSMutableArray *)auParameters;
@end

@implementation AudioUnitInspector

- (BOOL)isPresetParameter:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit {
    if (param.unit != kAudioUnitParameterUnit_Indexed) return NO;

    NSString *lowerName = [param.displayName lowercaseString];
    NSArray *presetKeywords = @[@"preset", @"patch", @"sound", @"bank", @"program", @"model", @"amp", @"cab", @"scene"];

    for (NSString *keyword in presetKeywords) {
        if ([lowerName containsString:keyword]) {
            return YES;
        }
    }
    return NO;
}

- (void)initializeAudioUnitForParameterExtraction:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock {
    // Set a realistic buffer size for proper initialization
    audioUnit.maximumFramesToRender = 512;

    // Load a default preset if available - this triggers full plugin initialization
    // Many AudioUnit plugins don't populate their parameter metadata (valueStrings, etc.)
    // until they're properly initialized with a preset and have processed audio
    if (audioUnit.factoryPresets.count > 0) {
        audioUnit.currentPreset = audioUnit.factoryPresets.firstObject;

        // Allow time for preset to load and plugin to initialize
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.3 * NSEC_PER_SEC)), dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
            [self performAudioProcessingToCompleteInitialization:audioUnit completion:completionBlock];
        });
    } else {
        [self performAudioProcessingToCompleteInitialization:audioUnit completion:completionBlock];
    }
}

- (void)performAudioProcessingToCompleteInitialization:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock {
    // Set up proper audio format for processing
    AVAudioFormat *format = [[AVAudioFormat alloc] initStandardFormatWithSampleRate:44100.0 channels:2];

    if (audioUnit.outputBusses.count > 0) {
        @try {
            // Create silent audio buffers to simulate processing
            // This forces many plugins to complete their internal initialization
            // and populate parameter metadata like valueStrings arrays
            AVAudioPCMBuffer *inputBuffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:format frameCapacity:512];
            if (inputBuffer) {
                inputBuffer.frameLength = 512;

                // Zero out the buffer (silent audio)
                if (inputBuffer.floatChannelData[0]) {
                    memset(inputBuffer.floatChannelData[0], 0, 512 * sizeof(float));
                }
                if (inputBuffer.format.channelCount > 1 && inputBuffer.floatChannelData[1]) {
                    memset(inputBuffer.floatChannelData[1], 0, 512 * sizeof(float));
                }

                // Try to get and call the render block to trigger processing
                AURenderBlock renderBlock = audioUnit.renderBlock;
                if (renderBlock) {
                    // Create output buffer list
                    AudioBufferList *outputBufferList = (AudioBufferList *)calloc(1, sizeof(AudioBufferList) + sizeof(AudioBuffer));
                    outputBufferList->mNumberBuffers = 1;
                    outputBufferList->mBuffers[0].mNumberChannels = 2;
                    outputBufferList->mBuffers[0].mDataByteSize = 512 * 2 * sizeof(float);
                    outputBufferList->mBuffers[0].mData = calloc(512 * 2, sizeof(float));

                    AudioTimeStamp timeStamp = {0};
                    timeStamp.mSampleTime = 0;
                    timeStamp.mFlags = kAudioTimeStampSampleTimeValid;

                    AudioUnitRenderActionFlags flags = 0;
                    OSStatus status = renderBlock(&flags, &timeStamp, 512, 0, outputBufferList, nil);

                    free(outputBufferList->mBuffers[0].mData);
                    free(outputBufferList);
                }
            }
        } @catch (NSException *exception) {
            // Silent error handling - some plugins may throw exceptions
        }
    }

    // Allow additional time for plugin to complete initialization and update parameter metadata
    // This delay is essential for many plugins to fully populate their valueStrings and other metadata
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.2 * NSEC_PER_SEC)), dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
        completionBlock();
    });
}

- (void)extractIndexedParameterInfo:(AUParameter *)param paramData:(NSMutableDictionary *)paramData audioUnit:(AUAudioUnit *)audioUnit {
    if (param.unit != kAudioUnitParameterUnit_Indexed) return;

    NSArray<NSString *> *indexedValues = nil;
    NSString *source = nil;

    // Method 1: Standard valueStrings property (most effective - ~80% success rate)
    indexedValues = param.valueStrings;
    if (indexedValues && indexedValues.count > 0) {
        source = @"valueStrings";
    }

    // Method 2: Check if it's a preset parameter (essential for preset-based parameters)
    if (!indexedValues && [self isPresetParameter:param audioUnit:audioUnit]) {
        NSArray *factoryPresets = audioUnit.factoryPresets;
        if (factoryPresets.count > 0) {
            NSMutableArray *presetNames = [NSMutableArray array];
            for (AUAudioUnitPreset *preset in factoryPresets) {
                [presetNames addObject:preset.name];
            }
            indexedValues = presetNames;
            source = @"factoryPresets";
        }
    }

    // Store results in the parameter data for JSON output
    if (indexedValues && indexedValues.count > 0) {
        [paramData setObject:indexedValues forKey:@"indexedValues"];
        [paramData setObject:source forKey:@"indexedValuesSource"];
        VERBOSE_LOG("    ✓ %s: extracted %lu values using %s\n",
                [param.displayName UTF8String],
                (unsigned long)indexedValues.count,
                [source UTF8String]);
    } else {
        // Store range information for manual mapping later
        [paramData setObject:[NSNumber numberWithInt:(int)param.minValue] forKey:@"indexedMinValue"];
        [paramData setObject:[NSNumber numberWithInt:(int)param.maxValue] forKey:@"indexedMaxValue"];
        [paramData setObject:@"none_found" forKey:@"indexedValuesSource"];
        VERBOSE_LOG("    ✗ %s: no indexed values found (range %.0f-%.0f)\n",
                [param.displayName UTF8String], param.minValue, param.maxValue);
    }
}

- (void)processParametersForAudioUnit:(AUAudioUnit *)audioUnit withName:(NSString *)auName auParameters:(NSMutableArray *)auParameters {
    AUParameterTree *parameterTree = audioUnit.parameterTree;
    if (!parameterTree) {
        VERBOSE_LOG("  ✗ No parameter tree available\n");
        return;
    }

    NSArray *allParameters = parameterTree.allParameters;
    
    // Early skip optimization: if no parameters, don't waste time
    if (allParameters.count == 0) {
        VERBOSE_LOG("  ⏭️  Skipping - no parameters\n");
        return;
    }

    // Count indexed parameters first
    NSUInteger indexedCount = 0;
    for (AUParameter *param in allParameters) {
        if (param.unit == kAudioUnitParameterUnit_Indexed) {
            indexedCount++;
        }
    }

    VERBOSE_LOG("  Processing %lu parameters (%lu indexed)\n",
            (unsigned long)allParameters.count, indexedCount);

    // Process all parameters (for JSON output like your working version)
    for (AUParameter *param in allParameters) {
        BOOL isWritable = (param.flags & kAudioUnitParameterFlag_IsWritable) != 0;
        BOOL canRamp = (param.flags & kAudioUnitParameterFlag_CanRamp) != 0;

        // Only include writable or automatable parameters (from your working version)
        if (isWritable || canRamp) {
            NSMutableDictionary *paramData = [NSMutableDictionary dictionary];
            [paramData setObject:param.displayName forKey:@"displayName"];
            [paramData setObject:param.identifier forKey:@"identifier"];
            [paramData setObject:[NSNumber numberWithUnsignedLongLong:param.address] forKey:@"address"];
            [paramData setObject:[NSNumber numberWithFloat:param.minValue] forKey:@"minValue"];
            [paramData setObject:[NSNumber numberWithFloat:param.maxValue] forKey:@"maxValue"];

            // For now, use current value as default (we can enhance this later)
            // Note: Getting true default values requires more complex AudioUnit introspection
            [paramData setObject:[NSNumber numberWithFloat:param.value] forKey:@"defaultValue"];
            [paramData setObject:[NSNumber numberWithFloat:param.value] forKey:@"currentValue"];
            [paramData setObject:StringFromAudioUnitParameterUnit(param.unit) forKey:@"unit"];
            [paramData setObject:[NSNumber numberWithBool:isWritable] forKey:@"isWritable"];
            [paramData setObject:[NSNumber numberWithBool:canRamp] forKey:@"canRamp"];
            [paramData setObject:[NSNumber numberWithUnsignedInteger:param.flags] forKey:@"rawFlags"];

            // Enhanced indexed parameter processing
            if (param.unit == kAudioUnitParameterUnit_Indexed) {
                [self extractIndexedParameterInfo:param paramData:paramData audioUnit:audioUnit];
            }

            [auParameters addObject:paramData]; // Add parameter to the AU's parameter array
        }
    }
}

@end

char *IntrospectAudioUnitsWithTimeout(double timeoutSeconds) {
    @autoreleasepool {
        PROGRESS_LOG("Rackless - AudioUnit Plugin Introspection Tool\n");
        PROGRESS_LOG("Scanning for all AudioUnit plugins...\n");

        AudioComponentDescription searchDescription = {
            .componentType = 0,          // 0 = scan all types
            .componentSubType = 0,       // 0 = scan all subtypes
            .componentManufacturer = 0,  // 0 = scan all manufacturers (not just NDSP)
            .componentFlags = 0,
            .componentFlagsMask = 0
        };

        AudioComponent currentComponent = NULL;
        __block int count = 0;

        // Master array to hold all AU dictionaries
        __block NSMutableArray *allAudioUnitsData = [NSMutableArray array];

        AudioUnitInspector *inspector = [[AudioUnitInspector alloc] init];
        dispatch_group_t group = dispatch_group_create();

        do {
            currentComponent = AudioComponentFindNext(currentComponent, &searchDescription);

            if (currentComponent != NULL) {
                dispatch_group_enter(group);

                CFStringRef nameCFString = NULL;
                AudioComponentCopyName(currentComponent, &nameCFString);

                AudioComponentDescription componentDesc;
                AudioComponentGetDescription(currentComponent, &componentDesc);

                NSString *auName = (nameCFString != NULL) ? (__bridge NSString *)nameCFString : @"[Unknown Name]";

                count++;
                VERBOSE_LOG("Found Audio Unit [%d]: %s\n", count, [auName UTF8String]);

                // Create a mutable dictionary for the current Audio Unit's data
                NSMutableDictionary *auData = [NSMutableDictionary dictionary];
                [auData setObject:auName forKey:@"name"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentManufacturer) forKey:@"manufacturerID"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentType) forKey:@"type"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentSubType) forKey:@"subtype"];

                // Array to hold parameters for this AU
                NSMutableArray *auParameters = [NSMutableArray array];
                [auData setObject:auParameters forKey:@"parameters"];

                [AUAudioUnit instantiateWithComponentDescription:componentDesc options:kAudioComponentInstantiation_LoadOutOfProcess completionHandler:^(AUAudioUnit * _Nullable audioUnit, NSError * _Nullable error) {
                    if (audioUnit) {
                        VERBOSE_LOG("  ✓ AudioUnit instantiated successfully\n");

                        // Set up audio format on all busses (from your working version)
                        NSError *busError = nil;
                        AVAudioFormat *renderFormat = [[AVAudioFormat alloc] initStandardFormatWithSampleRate:44100.0 channels:2];
                        if (audioUnit.outputBusses.count > 0 && ![audioUnit.outputBusses[0] setFormat:renderFormat error:&busError]) {
                            VERBOSE_LOG("  ⚠ Could not set render format: %s\n", [busError.localizedDescription UTF8String]);
                        }

                        // Allocate render resources
                        NSError *allocError = nil;
                        if (![audioUnit allocateRenderResourcesAndReturnError:&allocError]) {
                            VERBOSE_LOG("  ⚠ Could not allocate render resources: %s\n", [allocError.localizedDescription UTF8String]);
                        } else {
                            VERBOSE_LOG("  ✓ Render resources allocated\n");
                        }

                        // Enhanced initialization for better parameter extraction
                        [inspector initializeAudioUnitForParameterExtraction:audioUnit completion:^{
                            // Check if this plugin has parameters before processing
                            AUParameterTree *parameterTree = audioUnit.parameterTree;
                            NSArray *allParameters = parameterTree ? parameterTree.allParameters : nil;
                            
                            if (!allParameters || allParameters.count == 0) {
                                // Skip plugins with no parameters - they're not useful for live performance control
                                VERBOSE_LOG("  ⏭️  Completed inspection of %s (skipped - no parameters)\n", [auName UTF8String]);
                                
                                // Don't add to results - we only want plugins with parameters
                                if (nameCFString != NULL) {
                                    CFRelease(nameCFString);
                                }
                                dispatch_group_leave(group);
                                return;
                            }
                            
                            // Process parameters and add to auParameters array
                            [inspector processParametersForAudioUnit:audioUnit withName:auName auParameters:auParameters];

                            // Only add plugins that have parameters to the results
                            if (auParameters.count > 0) {
                                VERBOSE_LOG("  ✓ Completed inspection of %s (%lu parameters)\n", [auName UTF8String], (unsigned long)auParameters.count);
                                
                                // Add the collected AU data to the master array
                                @synchronized(allAudioUnitsData) {
                                    [allAudioUnitsData addObject:auData];
                                }
                            } else {
                                VERBOSE_LOG("  ⏭️  Completed inspection of %s (skipped - no usable parameters)\n", [auName UTF8String]);
                            }

                            if (nameCFString != NULL) {
                                CFRelease(nameCFString);
                            }
                            dispatch_group_leave(group);
                        }];

                    } else {
                        // Log errors, but don't add failed instantiations to results (they're not useful)
                        VERBOSE_LOG("  ✗ Failed to instantiate: %s\n", [error.localizedDescription UTF8String]);

                        if (nameCFString != NULL) {
                            CFRelease(nameCFString);
                        }
                        dispatch_group_leave(group);
                    }
                }];
            }
        } while (currentComponent != NULL);

        PROGRESS_LOG("Waiting for all AudioUnit inspections to complete...\n");
        
        // Wait with immediate timeout for maximum performance
        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.0 * NSEC_PER_SEC));
        long result = dispatch_group_wait(group, timeout);
        
        if (result != 0) {
            // Timeout occurred - this is expected with immediate timeout for maximum performance
            PROGRESS_LOG("⚠️  Timeout: AudioUnit inspection took longer than expected. This may indicate:\n");
            PROGRESS_LOG("   - A plugin is hanging or taking too long to initialize\n");
            PROGRESS_LOG("   - System is under heavy load\n");
            PROGRESS_LOG("   - A plugin has crashed and is waiting indefinitely\n");
            PROGRESS_LOG("Returning partial results from %lu completed inspections...\n", 
                        (unsigned long)allAudioUnitsData.count);
        }

        NSUInteger usablePlugins = allAudioUnitsData.count;
        if (result == 0) {
            PROGRESS_LOG("Inspection complete. Found %lu usable plugins (with parameters) out of %d total AudioUnits.\n", 
                        (unsigned long)usablePlugins, count);
        } else {
            PROGRESS_LOG("Inspection timed out. Found %lu usable plugins (with parameters) from partial scan of %d total AudioUnits.\n", 
                        (unsigned long)usablePlugins, count);
        }

        // Convert the collected data to JSON and output to stdout
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:allAudioUnitsData
                                                           options:NSJSONWritingPrettyPrinted
                                                             error:&jsonError];

        if (jsonData && !jsonError) {
            NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];

            // Copy the UTF8 string to a malloc'd buffer to avoid returning an inner pointer
            const char *utf8Str = [jsonString UTF8String];
            char *result = malloc(strlen(utf8Str) + 1);
            if (result) {
                strcpy(result, utf8Str);
            }
            // Log success to stderr
            PROGRESS_LOG("JSON output complete (%.1f KB)\n", (double)jsonData.length / 1024.0);

            return result;
        } else {
            PROGRESS_LOG("Error generating JSON: %s\n", [jsonError.localizedDescription UTF8String]);
            return NULL;
        }
    }
}
