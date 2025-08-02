#import <Foundation/Foundation.h>

// Global verbose logging flag
int g_verboseLogging = 0;

// Declare the function from audiounit_inspector.m
extern char *IntrospectAudioUnitsWithTimeout(double timeoutSeconds);

void printUsage(const char *programName) {
    printf("Rackless - AudioUnit Plugin Introspection Tool\n\n");
    printf("USAGE:\n");
    printf("  %s [options] [timeout_seconds]\n", programName);
    printf("  %s --help\n\n", programName);
    printf("ARGUMENTS:\n");
    printf("  timeout_seconds    Per-plugin timeout in seconds (default: 0.1)\n");
    printf("                     Examples: 0.001 (1ms), 0.01 (10ms), 0.1 (100ms), 1.0 (1s)\n\n");
    printf("OPTIONS:\n");
    printf("  -v, --verbose      Enable verbose logging (shows detailed plugin processing)\n");
    printf("  --help, -h         Show this help message\n\n");
    printf("DESCRIPTION:\n");
    printf("  Scans all AudioUnit plugins on the system and extracts their parameters.\n");
    printf("  Each plugin is processed individually with the specified timeout to prevent\n");
    printf("  hanging plugins from blocking the entire scan.\n\n");
    printf("  The tool outputs JSON data to stdout and progress information to stderr.\n\n");
    printf("EXAMPLES:\n");
    printf("  %s                 # Use default 100ms timeout\n", programName);
    printf("  %s -v              # Use default timeout with verbose logging\n", programName);
    printf("  %s 0.01            # Use 10ms timeout (faster)\n", programName);
    printf("  %s -v 0.1          # Use 100ms timeout with verbose logging\n", programName);
    printf("  %s 1.0 > plugins.json  # Use 1 second timeout, save to file\n", programName);
}

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // Default timeout is 100ms (0.1 seconds) for balanced performance and completeness
        double timeoutSeconds = 0.1;
        int argIndex = 1;
        
        // Parse command-line arguments
        while (argIndex < argc) {
            const char *arg = argv[argIndex];
            
            // Check for help flag
            if (strcmp(arg, "--help") == 0 || strcmp(arg, "-h") == 0) {
                printUsage(argv[0]);
                return 0;
            }
            
            // Check for verbose flag
            if (strcmp(arg, "--verbose") == 0 || strcmp(arg, "-v") == 0) {
                g_verboseLogging = 1;
                argIndex++;
                continue;
            }
            
            // Must be timeout value
            char *endPtr;
            timeoutSeconds = strtod(arg, &endPtr);
            
            // Check if the entire string was consumed and the value is valid
            if (*endPtr != '\0' || timeoutSeconds <= 0) {
                fprintf(stderr, "Error: Invalid timeout value '%s'\n", arg);
                fprintf(stderr, "Expected a positive number (in seconds).\n\n");
                printUsage(argv[0]);
                return 1;
            }
            
            argIndex++;
        }
        
        char *jsonResult = IntrospectAudioUnitsWithTimeout(timeoutSeconds);
        
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
