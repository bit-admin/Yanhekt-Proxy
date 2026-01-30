import { Crypto } from './crypto';
import { handleHealth } from './handlers/health';
import { handleStream } from './handlers/stream';
import { handleSegment } from './handlers/segment';

export interface Env {
  UPSTREAM_API: string;
  VIDEO_HOST: string;
  MAGIC_KEY: string;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    // Log request
    console.log(`${request.method} ${path}`);

    // CORS preflight handling
    if (request.method === 'OPTIONS') {
      return new Response(null, {
        status: 200,
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type, Authorization',
        },
      });
    }

    const crypto = new Crypto(env.MAGIC_KEY);

    // Route handling
    if (path === '/health') {
      return handleHealth();
    }

    if (path === '/stream') {
      return handleStream({
        request,
        crypto,
        upstreamAPI: env.UPSTREAM_API,
        magicKey: env.MAGIC_KEY,
        videoHost: env.VIDEO_HOST,
      });
    }

    if (path.startsWith('/ts/')) {
      const tsFileName = path.slice(4); // Remove '/ts/' prefix
      return handleSegment({
        request,
        crypto,
        upstreamAPI: env.UPSTREAM_API,
        magicKey: env.MAGIC_KEY,
        videoHost: env.VIDEO_HOST,
        tsFileName,
      });
    }

    // 404 for unknown routes
    return new Response('Not Found', {
      status: 404,
      headers: { 'Access-Control-Allow-Origin': '*' },
    });
  },
};
