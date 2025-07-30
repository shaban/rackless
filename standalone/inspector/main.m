#import <Foundation/Foundation.h>

// Declare the function from audiounit_inspector.m
extern char *IntrospectAudioUnits();

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        char *jsonResult = IntrospectAudioUnits();
        
        if (jsonResult) {
            printf("%s\n", jsonResult);
            free(jsonResult);
        } else {
            fprintf(stderr, "Failed to introspect AudioUnits\n");
            return 1;
        }
    }
    return 0;
}
