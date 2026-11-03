"""Generate the simple opaque recorder icon from fixed vector line geometry."""
import json
from pathlib import Path
import struct
import zlib

ROOT = Path(__file__).resolve().parents[1]
DEST = ROOT / 'ios/VehicleFlightRecorder/Resources/Assets.xcassets'


def main():
    size = 1024
    pixels = bytearray(bytes((16, 29, 46)) * (size * size))
    def line(start, end, color, width):
        x0, y0 = start
        x1, y1 = end
        steps = max(abs(x1-x0), abs(y1-y0))
        radius = width // 2
        for step in range(steps + 1):
            x = round(x0 + (x1-x0) * step / max(1, steps))
            y = round(y0 + (y1-y0) * step / max(1, steps))
            for py in range(max(0,y-radius), min(size,y+radius+1)):
                for px in range(max(0,x-radius), min(size,x+radius+1)):
                    if (px-x)**2 + (py-y)**2 <= radius**2:
                        offset = (py*size+px)*3
                        pixels[offset:offset+3] = bytes(color)
    # Car roof/body over a measured trace; all geometry is part of the source.
    outline = [(200,470),(270,445),(350,330),(650,330),(745,445),(825,475),(825,590),(200,590),(200,470)]
    for a,b in zip(outline, outline[1:]): line(a,b,(229,238,245),22)
    trace = [(190,720),(345,720),(410,645),(475,795),(545,665),(610,720),(830,720)]
    for a,b in zip(trace,trace[1:]): line(a,b,(68,213,166),24)
    def chunk(kind, payload):
        return struct.pack('>I',len(payload))+kind+payload+struct.pack('>I',zlib.crc32(kind+payload))
    scan = b''.join(b'\0'+pixels[row*size*3:(row+1)*size*3] for row in range(size))
    png = b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',size,size,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(scan))+chunk(b'IEND',b'')
    icon = DEST / 'AppIcon.appiconset'
    icon.mkdir(parents=True,exist_ok=True)
    (icon/'AppIcon.png').write_bytes(png)
    (DEST/'Contents.json').write_text(json.dumps({'info':{'author':'xcode','version':1}},indent=2)+'\n')
    (icon/'Contents.json').write_text(json.dumps({'images':[{'filename':'AppIcon.png','idiom':'universal','platform':'ios','size':'1024x1024'}], 'info':{'author':'xcode','version':1}},indent=2)+'\n')


if __name__ == '__main__':
    main()
