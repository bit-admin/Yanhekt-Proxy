import { md5 } from './md5';

interface VideoTokenResponse {
  code: number | string;
  message: string;
  data: {
    token: string;
  };
}

/**
 * Fetches video token from upstream API
 * No caching - fetches fresh token each request
 */
export async function fetchVideoToken(
  loginToken: string,
  upstreamAPI: string,
  magicKey: string
): Promise<string> {
  const url = `${upstreamAPI}/v1/auth/video/token?id=0`;

  const timestamp = Math.floor(Date.now() / 1000).toString();
  const signature = md5(magicKey + '_v1_undefined');

  const response = await fetch(url, {
    method: 'GET',
    headers: {
      Origin: 'https://www.yanhekt.cn',
      Referer: 'https://www.yanhekt.cn/',
      'User-Agent':
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.3',
      'xdomain-client': 'web_user',
      'Xdomain-Client': 'web_user',
      'Xclient-Version': 'v1',
      'Xclient-Signature': signature,
      'Xclient-Timestamp': timestamp,
      Authorization: `Bearer ${loginToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Token API returned status ${response.status}`);
  }

  const result: VideoTokenResponse = await response.json();

  // Check for success (code can be 0 or "0")
  const codeOK = result.code === 0 || result.code === '0';

  if (!codeOK) {
    throw new Error(`API error: ${result.message}`);
  }

  return result.data.token;
}
