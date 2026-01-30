const BASE_HEADERS: Record<string, string> = {
  Origin: 'https://www.yanhekt.cn',
  Referer: 'https://www.yanhekt.cn/',
  'User-Agent':
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.3',
};

const MAX_RETRIES = 3;

interface RetryOptions {
  getURL: () => string;
  videoHost: string;
  onRetry: () => Promise<void>;
}

/**
 * Fetches M3U8 content with retry logic for 403 errors
 */
export async function fetchM3U8WithRetry(
  options: RetryOptions
): Promise<string> {
  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    const url = options.getURL();

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: BASE_HEADERS,
      });

      if (response.ok) {
        return await response.text();
      }

      if (response.status === 403 && attempt < MAX_RETRIES) {
        lastError = new Error('M3U8 request got 403');
        await options.onRetry();
        continue;
      }

      throw new Error(`M3U8 request failed with status ${response.status}`);
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      if (attempt < MAX_RETRIES) {
        await options.onRetry();
        continue;
      }
    }
  }

  throw new Error(
    `M3U8 request failed after ${MAX_RETRIES} retries: ${lastError?.message}`
  );
}

/**
 * Fetches TS segment with retry logic for 403 errors
 */
export async function fetchTSWithRetry(
  options: RetryOptions
): Promise<Response> {
  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    const url = options.getURL();

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: BASE_HEADERS,
      });

      if (response.ok) {
        return response;
      }

      if (response.status === 403 && attempt < MAX_RETRIES) {
        lastError = new Error('TS request got 403');
        await options.onRetry();
        continue;
      }

      throw new Error(`TS request failed with status ${response.status}`);
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      if (attempt < MAX_RETRIES) {
        await options.onRetry();
        continue;
      }
    }
  }

  throw new Error(
    `TS request failed after ${MAX_RETRIES} retries: ${lastError?.message}`
  );
}
