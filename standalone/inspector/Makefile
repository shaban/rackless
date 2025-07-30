CC = clang
CFLAGS = -Wall -Wextra -std=c11 -fobjc-arc
FRAMEWORKS = -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit

TARGET = inspector
SOURCES = main.m audiounit_inspector.m

all: $(TARGET)

$(TARGET): $(SOURCES)
	$(CC) $(CFLAGS) $(FRAMEWORKS) -o $(TARGET) $(SOURCES)

clean:
	rm -f $(TARGET)

.PHONY: all clean
