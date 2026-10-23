"""Build a minimal AMD64 COFF icon resource from the original ICO (stdlib only)."""
import struct
from pathlib import Path

u16=lambda x:struct.pack('<H',x)
u32=lambda x:struct.pack('<I',x)
ico=Path('icon.ico').read_bytes()
count=struct.unpack_from('<H',ico,4)[0]
icons=[]
group=ico[:6]
for i in range(count):
    entry=ico[6+i*16:22+i*16]
    size,offset=struct.unpack_from('<II',entry,8)
    icons.append((i+1,ico[offset:offset+size]))
    group+=entry[:12]+u16(i+1)
manifest=b'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
<assemblyIdentity version="0.1.0.0" processorArchitecture="amd64" name="OPPO.P5.Repair" type="win32"/>
<trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo>
</assembly>'''
types=[(3,icons),(14,[(1,group)]),(24,[(1,manifest)])]
data=bytearray()
def reserve(n):
    pos=len(data);data.extend(bytes(n));return pos
def put(pos,b):data[pos:pos+len(b)]=b
def directory(entries):
    pos=reserve(16+8*len(entries));put(pos+14,u16(len(entries)));return pos
root=directory(types)
leaves=[]
for i,(typ,items) in enumerate(types):
    sub=directory(items);put(root+16+i*8,u32(typ)+u32(sub|0x80000000))
    for j,(ident,blob) in enumerate(items):
        lang=directory([0]);put(sub+16+j*8,u32(ident)+u32(lang|0x80000000))
        entry=reserve(16);put(lang+16,u32(0x409)+u32(entry));leaves.append((entry,blob))
relocs=[]
for entry,blob in leaves:
    reserve((-len(data))%4);offset=len(data);data.extend(blob)
    put(entry,u32(offset)+u32(len(blob))+bytes(8));relocs.append(struct.pack('<IIH',entry,0,3))
raw=60;reloc=raw+len(data);sym=reloc+10*len(relocs)
header=struct.pack('<HHIIIHH',0x8664,1,0,sym,1,0,0)
section=b'.rsrc\0\0\0'+struct.pack('<IIIIIIHHI',0,0,len(data),raw,reloc,0,len(relocs),0,0x40000040)
symbol=b'.rsrc\0\0\0'+struct.pack('<IhHBB',0,1,0,3,0)
Path('resource_windows_amd64.syso').write_bytes(header+section+data+b''.join(relocs)+symbol+u32(4))
print(f'Embedded {count} original icon images and asInvoker manifest')
