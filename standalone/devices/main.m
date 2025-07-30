#import <Foundation/Foundation.h>
#import "audiounit_devices.h"

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        NSMutableDictionary *systemDevices = [NSMutableDictionary dictionary];
        
        // Get all device types using the Archive functions
        char *audioInputStr = getAudioInputDevices();
        char *audioOutputStr = getAudioOutputDevices();
        char *defaultDevicesStr = getDefaultAudioDevices();
        char *midiInputStr = getMIDIInputDevices();
        char *midiOutputStr = getMIDIOutputDevices();
        
        // Parse JSON strings into objects
        NSError *error;
        NSData *data;
        
        // Audio Input Devices
        if (audioInputStr) {
            data = [NSData dataWithBytes:audioInputStr length:strlen(audioInputStr)];
            NSArray *audioInputDevices = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
            if (!error && audioInputDevices) {
                systemDevices[@"audioInput"] = audioInputDevices;
            }
            free(audioInputStr);
        }
        
        // Audio Output Devices  
        if (audioOutputStr) {
            data = [NSData dataWithBytes:audioOutputStr length:strlen(audioOutputStr)];
            NSArray *audioOutputDevices = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
            if (!error && audioOutputDevices) {
                systemDevices[@"audioOutput"] = audioOutputDevices;
            }
            free(audioOutputStr);
        }
        
        // Default Devices
        if (defaultDevicesStr) {
            data = [NSData dataWithBytes:defaultDevicesStr length:strlen(defaultDevicesStr)];
            NSDictionary *defaultDevices = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
            if (!error && defaultDevices) {
                systemDevices[@"defaults"] = defaultDevices;
            }
            free(defaultDevicesStr);
        }
        
        // MIDI Input Devices
        if (midiInputStr) {
            data = [NSData dataWithBytes:midiInputStr length:strlen(midiInputStr)];
            NSArray *midiInputDevices = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
            if (!error && midiInputDevices) {
                systemDevices[@"midiInput"] = midiInputDevices;
            }
            free(midiInputStr);
        }
        
        // MIDI Output Devices
        if (midiOutputStr) {
            data = [NSData dataWithBytes:midiOutputStr length:strlen(midiOutputStr)];
            NSArray *midiOutputDevices = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
            if (!error && midiOutputDevices) {
                systemDevices[@"midiOutput"] = midiOutputDevices;
            }
            free(midiOutputStr);
        }
        
        // Add metadata
        systemDevices[@"timestamp"] = [[NSDate date] description];
        systemDevices[@"totalAudioInputDevices"] = @([systemDevices[@"audioInput"] count]);
        systemDevices[@"totalAudioOutputDevices"] = @([systemDevices[@"audioOutput"] count]);
        systemDevices[@"totalMIDIInputDevices"] = @([systemDevices[@"midiInput"] count]);
        systemDevices[@"totalMIDIOutputDevices"] = @([systemDevices[@"midiOutput"] count]);
        
        // Output unified JSON to stdout
        NSError *jsonError;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:systemDevices 
                                                           options:NSJSONWritingPrettyPrinted 
                                                             error:&jsonError];
        
        if (jsonData && !jsonError) {
            NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            printf("%s\n", [jsonString UTF8String]);
        } else {
            fprintf(stderr, "Error generating JSON: %s\n", [jsonError.localizedDescription UTF8String]);
            return 1;
        }
    }
    
    return 0;
}
