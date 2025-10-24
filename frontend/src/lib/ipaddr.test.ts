import { expect, test, describe } from 'vitest';
import { IPAddress, CIDR } from './ipaddr';

test('parses IPv4 addresses correctly', () => {
  expect(IPAddress.parse("10.0.0.1").octets).toEqual([10, 0, 0, 1]);
  expect(IPAddress.parse("192.168.0.2").octets).toEqual([192, 168, 0, 2]);
});

test('throws error for invalid IP addresses', () => {
  expect(() => IPAddress.parse("256.0.0.1")).toThrow('Invalid IP address part: 256');
  expect(() => IPAddress.parse("10.0.-1.1")).toThrow('Invalid IP address part: -1');
  expect(() => IPAddress.parse("10.0.0")).toThrow('Invalid IP address string');
  expect(() => IPAddress.parse("abc.def.ghi.jkl")).toThrow('Invalid IP address part: abc');
  expect(() => IPAddress.parse("192.0.0.1/30")).toThrow('Invalid IP address part: 1/30');
});

test('converts IPAddress back to string correctly', () => {
  expect(IPAddress.parse("192.168.1.2").toString()).toEqual("192.168.1.2");
});

test('parses CIDR addresses correctly', () => {
  const cidr = CIDR.parse("10.0.0.1/8");
  expect(cidr.ip.octets).toEqual([10, 0, 0, 1]);
  expect(cidr.mask).toEqual(8);

  const cidr2 = CIDR.parse("192.168.0.2/24");
  expect(cidr2.ip.octets).toEqual([192, 168, 0, 2]);
  expect(cidr2.mask).toEqual(24);
});

test('throws error for invalid CIDR addresses', () => {
  expect(() => CIDR.parse("256.0.0.1/24")).toThrow('Invalid IP address part: 256');
  expect(() => CIDR.parse("10.0.0.1/33")).toThrow('Invalid CIDR mask');
  expect(() => CIDR.parse("10.0.0.1")).toThrow('Invalid CIDR string');
  expect(() => CIDR.parse("10.0.0.1/24/12")).toThrow('Invalid CIDR string');
});

test('checks if IP is in CIDR range', () => {
  const cidr1 = CIDR.parse("192.168.1.0/24");
  expect(cidr1.contains(IPAddress.parse("192.168.1.100"))).toBe(true);
  expect(cidr1.contains(IPAddress.parse("192.168.1.255"))).toBe(true);
  expect(cidr1.contains(IPAddress.parse("192.168.2.1"))).toBe(false);

  const cidr2 = CIDR.parse("10.0.0.0/8");
  expect(cidr2.contains(IPAddress.parse("10.1.2.3"))).toBe(true);
  expect(cidr2.contains(IPAddress.parse("11.0.0.1"))).toBe(false);

  const cidr3 = CIDR.parse("172.16.0.0/12");
  expect(cidr3.contains(IPAddress.parse("172.16.10.20"))).toBe(true);
  expect(cidr3.contains(IPAddress.parse("172.31.255.255"))).toBe(true);
  expect(cidr3.contains(IPAddress.parse("172.32.0.0"))).toBe(false);
});

describe('IPAddress comparison', () => {
  test('compareTo returns 0 for equal IP addresses', () => {
    const ip1 = IPAddress.parse("192.168.1.1");
    const ip2 = IPAddress.parse("192.168.1.1");
    expect(ip1.compareTo(ip2)).toBe(0);
  });

  test('compareTo returns negative when first IP is smaller', () => {
    const ip1 = IPAddress.parse("192.168.1.1");
    const ip2 = IPAddress.parse("192.168.1.2");
    expect(ip1.compareTo(ip2)).toBeLessThan(0);
  });

  test('compareTo returns positive when first IP is greater', () => {
    const ip1 = IPAddress.parse("192.168.1.2");
    const ip2 = IPAddress.parse("192.168.1.1");
    expect(ip1.compareTo(ip2)).toBeGreaterThan(0);
  });

  test('compareTo works across different octets', () => {
    const ip1 = IPAddress.parse("192.168.1.255");
    const ip2 = IPAddress.parse("192.168.2.0");
    expect(ip1.compareTo(ip2)).toBeLessThan(0);

    const ip3 = IPAddress.parse("192.167.255.255");
    const ip4 = IPAddress.parse("192.168.0.0");
    expect(ip3.compareTo(ip4)).toBeLessThan(0);

    const ip5 = IPAddress.parse("10.0.0.0");
    const ip6 = IPAddress.parse("192.168.1.1");
    expect(ip5.compareTo(ip6)).toBeLessThan(0);
  });

  test('equals returns true for identical IPs', () => {
    const ip1 = IPAddress.parse("192.168.1.1");
    const ip2 = IPAddress.parse("192.168.1.1");
    expect(ip1.equals(ip2)).toBe(true);
  });

  test('equals returns false for different IPs', () => {
    const ip1 = IPAddress.parse("192.168.1.1");
    const ip2 = IPAddress.parse("192.168.1.2");
    expect(ip1.equals(ip2)).toBe(false);
  });

  test('compareTo handles boundary values', () => {
    const minIP = IPAddress.parse("0.0.0.0");
    const maxIP = IPAddress.parse("255.255.255.255");
    expect(minIP.compareTo(maxIP)).toBeLessThan(0);
    expect(maxIP.compareTo(minIP)).toBeGreaterThan(0);
  });

  test('compareTo is transitive', () => {
    const ip1 = IPAddress.parse("10.0.0.1");
    const ip2 = IPAddress.parse("10.0.0.2");
    const ip3 = IPAddress.parse("10.0.0.3");

    expect(ip1.compareTo(ip2)).toBeLessThan(0);
    expect(ip2.compareTo(ip3)).toBeLessThan(0);
    expect(ip1.compareTo(ip3)).toBeLessThan(0);
  });

  test('compareTo is symmetric', () => {
    const ip1 = IPAddress.parse("172.16.0.1");
    const ip2 = IPAddress.parse("172.16.0.2");

    expect(Math.sign(ip1.compareTo(ip2))).toBe(-Math.sign(ip2.compareTo(ip1)));
  });

  test('can sort array of IP addresses', () => {
    const ips = [
      IPAddress.parse("192.168.1.10"),
      IPAddress.parse("10.0.0.1"),
      IPAddress.parse("192.168.1.5"),
      IPAddress.parse("172.16.0.1"),
      IPAddress.parse("192.168.1.1"),
    ];

    ips.sort((a, b) => a.compareTo(b));

    expect(ips[0].toString()).toBe("10.0.0.1");
    expect(ips[1].toString()).toBe("172.16.0.1");
    expect(ips[2].toString()).toBe("192.168.1.1");
    expect(ips[3].toString()).toBe("192.168.1.5");
    expect(ips[4].toString()).toBe("192.168.1.10");
  });

  test('compareTo with same reference', () => {
    const ip = IPAddress.parse("192.168.1.1");
    expect(ip.compareTo(ip)).toBe(0);
    expect(ip.equals(ip)).toBe(true);
  });
});
