import { md5 } from './md5';

/**
 * Crypto utilities for URL encryption and signing
 */
export class Crypto {
  constructor(private magicKey: string) {}

  /**
   * Inserts MD5 hash before the last path segment
   * Example: https://cvideo.yanhekt.cn/path/to/file.ts
   *       -> https://cvideo.yanhekt.cn/path/to/<hash>/file.ts
   */
  encryptURL(url: string): string {
    const parts = url.split('/');
    if (parts.length < 2) {
      return url;
    }

    const hash = md5(this.magicKey + '_100');

    // Insert hash before the last segment
    const lastIdx = parts.length - 1;
    const result = [...parts.slice(0, lastIdx), hash, parts[lastIdx]];

    return result.join('/');
  }

  /**
   * Generates timestamp and MD5 signature
   */
  getSignature(): { timestamp: string; signature: string } {
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const signature = md5(this.magicKey + '_v1_' + timestamp);
    return { timestamp, signature };
  }

  /**
   * Appends all authentication parameters to URL
   */
  signURL(url: string, videoToken: string): string {
    const { timestamp, signature } = this.getSignature();
    return `${url}?Xvideo_Token=${videoToken}&Xclient_Timestamp=${timestamp}&Xclient_Signature=${signature}&Xclient_Version=v1&Platform=yhkt_user`;
  }
}
