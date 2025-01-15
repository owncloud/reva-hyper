<?php declare(strict_types=1);
/**
 * ownCloud
 *
 * @author Nabin Magar <nabin@jankaritech.com>
 * @copyright Copyright (c) 2025 Nabin Magar nabin@jankaritech.com
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License,
 * as published by the Free Software Foundation;
 * either version 3 of the License, or any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>
 *
 */

namespace TestHelpers;

use Carbon\Carbon;
use GuzzleHttp\Exception\ClientException;
use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use TusPhp\Exception\ConnectionException;
use TusPhp\Exception\FileException;
use Psr\Http\Message\ResponseInterface;
use Symfony\Component\HttpFoundation\Response as HttpResponse;
use TusPhp\Exception\TusException;
use TusPhp\Tus\Client;

/**
 * A TUS client based on TusPhp\Tus\Client
 */

class TusClient extends Client {

	/**
	 * @param string $key
	 * @param int $bytes
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function createUploadWithResponse(string $key, int $bytes = -1): ResponseInterface {
		$bytes = $bytes < 0 ? $this->fileSize : $bytes;
		$headers = $this->headers + [
				'Upload-Length' => $this->fileSize,
				'Upload-Key' => $key,
				'Upload-Checksum' => $this->getUploadChecksumHeader(),
				'Upload-Metadata' => $this->getUploadMetadataHeader(),
			];
		$data = '';
		if ($bytes > 0) {
			$data = $this->getData(0, $bytes);

			$headers += [
				'Content-Type' => self::HEADER_CONTENT_TYPE,
				'Content-Length' => \strlen($data),
			];
		}
		if ($this->isPartial()) {
			$headers += ['Upload-Concat' => 'partial'];
		}
		try {
			$response = $this->getClient()->post(
				$this->apiPath,
				[
				'body' => $data,
				'headers' => $headers,
				]
			);
		} catch (ClientException $e) {
			$response = $e->getResponse();
		}
		$statusCode = $response->getStatusCode();
		if ($statusCode !== HttpResponse::HTTP_CREATED) {
			return $response;
		}
		$uploadLocation = current($response->getHeader('location'));
		$this->getCache()->set(
			$this->getKey(),
			[
			'location' => $uploadLocation,
			'expires_at' => Carbon::now()->addSeconds($this->getCache()->getTtl())->format($this->getCache()::RFC_7231),
			]
		);
		return $response;
	}

	/**
	 * @param int $bytes
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws TusException | ConnectionException
	 */
	public function uploadWithResponse(int $bytes = -1): ResponseInterface {
		$bytes  = $bytes < 0 ? $this->getFileSize() : $bytes;
		$offset = $this->partialOffset < 0 ? 0 : $this->partialOffset;
		try {
			// Check if this upload exists with HEAD request.
			$offset = $this->sendHeadRequest();
		} catch (FileException | ClientException $e) {
			// Create a new upload.
			$this->url = $this->createUploadWithResponse($this->getKey(), 0);
			if ($this->url->getStatusCode() !== HttpResponse::HTTP_CREATED) {
				return $this->url;
			}
		}
		// Verify that upload is not yet expired.
		if ($this->isExpired()) {
			throw new TusException('Upload expired.');
		}
		$data = $this->getData($offset, $bytes);
		$headers = $this->headers + [
				'Content-Type' => self::HEADER_CONTENT_TYPE,
				'Content-Length' => \strlen($data),
				'Upload-Checksum' => $this->getUploadChecksumHeader(),
			];
		if ($this->isPartial()) {
			$headers += ['Upload-Concat' => self::UPLOAD_TYPE_PARTIAL];
		} else {
			$headers += ['Upload-Offset' => $offset];
		}
		try {
			$response = $this->getClient()->patch(
				$this->getUrl(),
				[
				'body' => $data,
				'headers' => $headers,
				]
			);
		} catch (ClientException $e) {
			throw $this->handleClientException($e);
		}
		return $response;
	}
}
