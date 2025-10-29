import { expect, test, describe } from 'vitest'
import { IPAddress, CIDR } from './ipaddr'

test('parses IPv4 addresses correctly', () => {
  expect(IPAddress.parse('10.0.0.1').octets).toEqual([10, 0, 0, 1])
  expect(IPAddress.parse('192.168.0.2').octets).toEqual([192, 168, 0, 2])
})

test('throws error for invalid IP addresses', () => {
  expect(() => IPAddress.parse('256.0.0.1')).toThrow('Invalid IP address part: 256')
  expect(() => IPAddress.parse('10.0.-1.1')).toThrow('Invalid IP address part: -1')
  expect(() => IPAddress.parse('10.0.0')).toThrow('Invalid IP address string')
  expect(() => IPAddress.parse('abc.def.ghi.jkl')).toThrow('Invalid IP address part: abc')
  expect(() => IPAddress.parse('192.0.0.1/30')).toThrow('Invalid IP address part: 1/30')
})

test('converts IPAddress back to string correctly', () => {
  expect(IPAddress.parse('192.168.1.2').toString()).toEqual('192.168.1.2')
})

test('parses CIDR addresses correctly', () => {
  const cidr = CIDR.parse('10.0.0.1/8')
  expect(cidr.ip.octets).toEqual([10, 0, 0, 1])
  expect(cidr.mask).toEqual(8)

  const cidr2 = CIDR.parse('192.168.0.2/24')
  expect(cidr2.ip.octets).toEqual([192, 168, 0, 2])
  expect(cidr2.mask).toEqual(24)
})

test('throws error for invalid CIDR addresses', () => {
  expect(() => CIDR.parse('256.0.0.1/24')).toThrow('Invalid IP address part: 256')
  expect(() => CIDR.parse('10.0.0.1/33')).toThrow('Invalid CIDR mask')
  expect(() => CIDR.parse('10.0.0.1')).toThrow('Invalid CIDR string')
  expect(() => CIDR.parse('10.0.0.1/24/12')).toThrow('Invalid CIDR string')
})

test('checks if IP is in CIDR range', () => {
  const cidr1 = CIDR.parse('192.168.1.0/24')
  expect(cidr1.contains(IPAddress.parse('192.168.1.100'))).toBe(true)
  expect(cidr1.contains(IPAddress.parse('192.168.1.255'))).toBe(true)
  expect(cidr1.contains(IPAddress.parse('192.168.2.1'))).toBe(false)

  const cidr2 = CIDR.parse('10.0.0.0/8')
  expect(cidr2.contains(IPAddress.parse('10.1.2.3'))).toBe(true)
  expect(cidr2.contains(IPAddress.parse('11.0.0.1'))).toBe(false)

  const cidr3 = CIDR.parse('172.16.0.0/12')
  expect(cidr3.contains(IPAddress.parse('172.16.10.20'))).toBe(true)
  expect(cidr3.contains(IPAddress.parse('172.31.255.255'))).toBe(true)
  expect(cidr3.contains(IPAddress.parse('172.32.0.0'))).toBe(false)
})

describe('IPAddress comparison', () => {
  test('compareTo returns 0 for equal IP addresses', () => {
    const ip1 = IPAddress.parse('192.168.1.1')
    const ip2 = IPAddress.parse('192.168.1.1')
    expect(ip1.compareTo(ip2)).toBe(0)
  })

  test('compareTo returns negative when first IP is smaller', () => {
    const ip1 = IPAddress.parse('192.168.1.1')
    const ip2 = IPAddress.parse('192.168.1.2')
    expect(ip1.compareTo(ip2)).toBeLessThan(0)
  })

  test('compareTo returns positive when first IP is greater', () => {
    const ip1 = IPAddress.parse('192.168.1.2')
    const ip2 = IPAddress.parse('192.168.1.1')
    expect(ip1.compareTo(ip2)).toBeGreaterThan(0)
  })

  test('compareTo works across different octets', () => {
    const ip1 = IPAddress.parse('192.168.1.255')
    const ip2 = IPAddress.parse('192.168.2.0')
    expect(ip1.compareTo(ip2)).toBeLessThan(0)

    const ip3 = IPAddress.parse('192.167.255.255')
    const ip4 = IPAddress.parse('192.168.0.0')
    expect(ip3.compareTo(ip4)).toBeLessThan(0)

    const ip5 = IPAddress.parse('10.0.0.0')
    const ip6 = IPAddress.parse('192.168.1.1')
    expect(ip5.compareTo(ip6)).toBeLessThan(0)
  })

  test('equals returns true for identical IPs', () => {
    const ip1 = IPAddress.parse('192.168.1.1')
    const ip2 = IPAddress.parse('192.168.1.1')
    expect(ip1.equals(ip2)).toBe(true)
  })

  test('equals returns false for different IPs', () => {
    const ip1 = IPAddress.parse('192.168.1.1')
    const ip2 = IPAddress.parse('192.168.1.2')
    expect(ip1.equals(ip2)).toBe(false)
  })

  test('compareTo handles boundary values', () => {
    const minIP = IPAddress.parse('0.0.0.0')
    const maxIP = IPAddress.parse('255.255.255.255')
    expect(minIP.compareTo(maxIP)).toBeLessThan(0)
    expect(maxIP.compareTo(minIP)).toBeGreaterThan(0)
  })

  test('compareTo is transitive', () => {
    const ip1 = IPAddress.parse('10.0.0.1')
    const ip2 = IPAddress.parse('10.0.0.2')
    const ip3 = IPAddress.parse('10.0.0.3')

    expect(ip1.compareTo(ip2)).toBeLessThan(0)
    expect(ip2.compareTo(ip3)).toBeLessThan(0)
    expect(ip1.compareTo(ip3)).toBeLessThan(0)
  })

  test('compareTo is symmetric', () => {
    const ip1 = IPAddress.parse('172.16.0.1')
    const ip2 = IPAddress.parse('172.16.0.2')

    expect(Math.sign(ip1.compareTo(ip2))).toBe(-Math.sign(ip2.compareTo(ip1)))
  })

  test('can sort array of IP addresses', () => {
    const ips = [
      IPAddress.parse('192.168.1.10'),
      IPAddress.parse('10.0.0.1'),
      IPAddress.parse('192.168.1.5'),
      IPAddress.parse('172.16.0.1'),
      IPAddress.parse('192.168.1.1'),
    ]

    ips.sort((a, b) => a.compareTo(b))

    expect(ips[0]!.toString()).toBe('10.0.0.1')
    expect(ips[1]!.toString()).toBe('172.16.0.1')
    expect(ips[2]!.toString()).toBe('192.168.1.1')
    expect(ips[3]!.toString()).toBe('192.168.1.5')
    expect(ips[4]!.toString()).toBe('192.168.1.10')
  })

  test('compareTo with same reference', () => {
    const ip = IPAddress.parse('192.168.1.1')
    expect(ip.compareTo(ip)).toBe(0)
    expect(ip.equals(ip)).toBe(true)
  })
})

describe('IPAddress network, gateway and broadcast checks', () => {
  test('check if address is network address', () => {
    expect(CIDR.parse('192.168.0.0/24').isNetworkAddr()).toBe(true)
    expect(CIDR.parse('192.168.0.0/30').isNetworkAddr()).toBe(true)
    expect(CIDR.parse('10.0.0.0/30').isNetworkAddr()).toBe(true)
    expect(CIDR.parse('10.0.0.0/8').isNetworkAddr()).toBe(true)
    expect(CIDR.parse('10.0.0.128/26').isNetworkAddr()).toBe(true)
    expect(CIDR.parse('172.32.0.192/27').isNetworkAddr()).toBe(true)

    expect(CIDR.parse('192.168.1.2/24').isNetworkAddr()).toBe(false)
    expect(CIDR.parse('192.168.0.3/30').isNetworkAddr()).toBe(false)
    expect(CIDR.parse('10.0.0.2/30').isNetworkAddr()).toBe(false)
    expect(CIDR.parse('10.1.0.0/8').isNetworkAddr()).toBe(false)
    expect(CIDR.parse('10.0.0.138/26').isNetworkAddr()).toBe(false)
    expect(CIDR.parse('172.32.0.202/27').isNetworkAddr()).toBe(false)
  })

  test('check network address', () => {
    expect(CIDR.parse('192.168.0.125/24').networkAddr().toString()).toBe('192.168.0.0')
    expect(CIDR.parse('192.168.0.243/30').networkAddr().toString()).toBe('192.168.0.240')
    expect(CIDR.parse('10.0.0.15/27').networkAddr().toString()).toBe('10.0.0.0')
    expect(CIDR.parse('172.32.0.192/31').networkAddr().toString()).toBe('172.32.0.192')
  })

  test('check if address is broadcast address', () => {
    expect(CIDR.parse('192.168.0.255/24').isBroadcastAddr()).toBe(true)
    expect(CIDR.parse('192.168.0.3/30').isBroadcastAddr()).toBe(true)
    expect(CIDR.parse('10.0.0.3/30').isBroadcastAddr()).toBe(true)
    expect(CIDR.parse('10.255.255.255/8').isBroadcastAddr()).toBe(true)
    expect(CIDR.parse('10.0.0.191/26').isBroadcastAddr()).toBe(true)
    expect(CIDR.parse('172.32.0.223/27').isBroadcastAddr()).toBe(true)

    expect(CIDR.parse('192.168.0.245/24').isBroadcastAddr()).toBe(false)
    expect(CIDR.parse('192.168.0.2/30').isBroadcastAddr()).toBe(false)
    expect(CIDR.parse('10.0.0.2/30').isBroadcastAddr()).toBe(false)
    expect(CIDR.parse('10.255.245.255/8').isBroadcastAddr()).toBe(false)
    expect(CIDR.parse('10.0.0.71/26').isBroadcastAddr()).toBe(false)
    expect(CIDR.parse('172.32.0.203/27').isBroadcastAddr()).toBe(false)
  })

  test('check broadcast address', () => {
    expect(CIDR.parse('192.168.0.0/24').broadcastAddr().toString()).toBe('192.168.0.255')
    expect(CIDR.parse('192.168.0.240/30').broadcastAddr().toString()).toBe('192.168.0.243')
    expect(CIDR.parse('10.0.0.0/27').broadcastAddr().toString()).toBe('10.0.0.31')
    expect(CIDR.parse('172.32.0.192/31').broadcastAddr().toString()).toBe('172.32.0.193')
    expect(CIDR.parse('172.32.0.192/32').broadcastAddr().toString()).toBe('172.32.0.192')
  })

  test('check hostMin and hostMax', () => {
    // /24
    let cidr = CIDR.parse('192.168.0.0/24')
    expect(cidr.minHostAddr().toString()).toBe('192.168.0.1')
    expect(cidr.maxHostAddr().toString()).toBe('192.168.0.254')
    // /30
    cidr = CIDR.parse('192.168.0.240/30')
    expect(cidr.minHostAddr().toString()).toBe('192.168.0.241')
    expect(cidr.maxHostAddr().toString()).toBe('192.168.0.242')
    // /31 (point-to-point, only 2 hosts, both network and broadcast are usable)
    cidr = CIDR.parse('10.0.0.0/31')
    expect(cidr.minHostAddr().toString()).toBe('10.0.0.0')
    expect(cidr.maxHostAddr().toString()).toBe('10.0.0.1')
    // /32 (single host)
    cidr = CIDR.parse('10.0.0.15/32')
    expect(cidr.minHostAddr().toString()).toBe('10.0.0.15')
    expect(cidr.maxHostAddr().toString()).toBe('10.0.0.15')
  })
})
