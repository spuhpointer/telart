
SOURCES = playbacksv.c
TARGET  = playbacksv
OBJECTS = $(SOURCES:%.c=%.o)
CFLAGS	= -I../ubftab -g
LIBS = -latmisrvinteg -latmi -lubf -lnstd -lpthread -lrt -ldl -lm

default: target

target: $(TARGET)

$(TARGET): $(OBJECTS)
	$(CC) -o $@ $(LDFLAGS) $^ $(LIBS)

%.o: %.c
	$(CC) -c -o $@ $< $(CFLAGS)

clean:
	-rm *.o
	-rm $(TARGET)

