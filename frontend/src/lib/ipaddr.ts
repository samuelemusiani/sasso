export class IPAddress {
  readonly octets: [number, number, number, number];

  constructor(octets: number[]) {
    if (octets.length !== 4 || octets.some(o => isNaN(o) || o < 0 || o > 255)) {
      throw new Error('Invalid octets for IP address');
    }
    this.octets = [octets[0]!, octets[1]!, octets[2]!, octets[3]!];
  }

  static parse(ip: string): IPAddress {
    if (ip === undefined || ip === null || ip === '') {
      throw new Error('Invalid IP address string');
    }
    const parts = ip.split('.');
    if (parts.length !== 4) {
      throw new Error('Invalid IP address string');
    }
    const octets = parts.map(part => {
      const num = Number(part);
      if (isNaN(num) || num < 0 || num > 255) {
        throw new Error(`Invalid IP address part: ${part}`);
      }
      return num;
    });
    return new IPAddress(octets);
  }

  toNumber(): number {
    return (
      (this.octets[0] << 24) |
      (this.octets[1] << 16) |
      (this.octets[2] << 8) |
      this.octets[3]
    ) >>> 0; // Ensure unsigned
  }

  toString(): string {
    return this.octets.join('.');
  }

  compareTo(other: IPAddress): number {
    const thisNum = this.toNumber();
    const otherNum = other.toNumber();
    const diff = thisNum - otherNum;
    return diff < 0 ? -1 : diff > 0 ? 1 : 0;
  }

  equals(other: IPAddress): boolean {
    return this.compareTo(other) === 0;
  }
}

export class CIDR {
  readonly ip: IPAddress;
  readonly mask: number;

  constructor(ip: IPAddress, mask: number) {
    if (isNaN(mask) || mask < 0 || mask > 32) {
      throw new Error('Invalid CIDR mask');
    }
    this.ip = ip;
    this.mask = mask;
  }

  static parse(cidr: string): CIDR {
    if (cidr === undefined || cidr === null || cidr === '') {
      throw new Error('Invalid CIDR string');
    }
    const parts = cidr.split('/');
    if (parts.length !== 2) {
      throw new Error('Invalid CIDR string');
    }
    const ip = IPAddress.parse(parts[0]!);
    const mask = Number(parts[1]);

    return new CIDR(ip, mask);
  }

  contains(ip: IPAddress): boolean {
    const mask = -1 << (32 - this.mask);
    return (ip.toNumber() & mask) === (this.ip.toNumber() & mask);
  }
}
