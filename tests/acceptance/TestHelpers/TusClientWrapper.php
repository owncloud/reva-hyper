<?php declare(strict_types=1);
/**
 * ownCloud
 *
 * @author Nabin Magar <nabin@jankaritech.com>
 * @copyright Copyright (c) 2017 Nabin Magar nabin@jankaritech.com
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
use TusPhp\Exception\FileException;
use GuzzleHttp\Exception\ClientException;
use Psr\Http\Message\ResponseInterface;
use GuzzleHttp\Exception\GuzzleException;
use Symfony\Component\HttpFoundation\Response as HttpResponse;
use ReflectionException;
use TusPhp\Tus\Client;

/**
 * A helper class where TUS is wrapped and done API requests
 */

class TusClientWrapper extends Client {
	public $client;

	/**
	 * Constructor for the TusClientWrapper.
	 *
	 * @param string $baseUri
	 * @param array $options
	 *
	 * @throws ReflectionException
	 */
	public function __construct(string $baseUri, array $options = []) {
		parent::__construct($baseUri, $options);
		$this->client = new Client($baseUri, $options);
	}

	/**
	 * @param string $key
	 *
	 * @return self
	 */
	public function setKey(string $key): self {
		$this->client->setKey($key);
		return $this;
	}

	/**
	 * @param string $file
	 * @param string|null $name
	 *
	 * @return self
	 */
	public function file(string $file, string $name = null): self {
		$this->client->file($file, $name);
		return $this;
	}

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
				'Upload-Length' => $this->client->fileSize,
				'Upload-Key' => $key,
				'Upload-Checksum' => $this->client->getUploadChecksumHeader(),
				'Upload-Metadata' => $this->client->getUploadMetadataHeader(),
			];

		$data = '';
		if ($bytes > 0) {
			$data = $this->client->getData(0, $bytes);

			$headers += [
				'Content-Type' => self::HEADER_CONTENT_TYPE,
				'Content-Length' => \strlen($data),
			];
		}

		if ($this->isPartial()) {
			$headers += ['Upload-Concat' => 'partial'];
		}

		try {
			$response = $this->client->getClient()->post(
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
			throw new FileException('Unable to create resource.');
		}

		$uploadLocation = current($response->getHeader('location'));

		$this->getCache()->set(
			$this->client->getKey(),
			[
			'location' => $uploadLocation,
			'expires_at' => Carbon::now()->addSeconds($this->getCache()->getTtl())->format($this->getCache()::RFC_7231),
			]
		);
		return $response;
	}
}
