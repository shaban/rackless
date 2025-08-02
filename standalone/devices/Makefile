CC = clang
CFLAGS = -Wall -Wextra -std=c11 -fobjc-arc
FRAMEWORKS = -framework Foundation -framework CoreAudio -framework CoreMIDI

TARGET = devices
SOURCES = main.m audiounit_devices.m

all: $(TARGET)

$(TARGET): $(SOURCES)
	$(CC) $(CFLAGS) $(FRAMEWORKS) -o $(TARGET) $(SOURCES)

clean:
	rm -f $(TARGET)

.PHONY: all clean
