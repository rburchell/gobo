#include "stream.h"

int main(int argc, char **argv) {
    (void)argc;
    (void)argv;

    PlaybackHeader header;
    header.setMagic(1010);
    header.setTestfloat(1.234);
    header.setTestdouble(5.678);

    PlaybackFile pf;
    pf.setHeader(header);
    pf.setBody("hello world");

    std::vector<uint8_t> data = encode(pf);
    FILE* f = fopen("stream.out", "wb");
    if (f == nullptr) {
        perror("open");
        exit(1);
    }
    fwrite(data.data(), data.size(), 1, f);
    fclose(f);

    {
        PlaybackFile pf2;
        decode(data, pf2);
        printf("Read: %d %s %f %f\n", pf2.header().magic(), pf2.body().data(), pf2.header().testfloat(), pf2.header().testdouble());
    }
}
