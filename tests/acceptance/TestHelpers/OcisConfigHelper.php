<?php declare(strict_types=1);
/**
 * ownCloud
 *
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2023 Sajan Gurung sajan@jankaritech.com
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

use GuzzleHttp\Exception\ConnectException;
use TestHelpers\HttpRequestHelper;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Psr7\Request;
use Psr\Http\Message\ResponseInterface;

/**
 * A helper class for configuring oCIS server
 */
class OcisConfigHelper {
	/**
	 * @param string $url
	 * @param string $method
	 * @param ?string $body
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function sendRequest(
		string $url,
		string $method,
		?string $body = ""
	): ResponseInterface {
		$client = HttpRequestHelper::createClient();
		$request = new Request(
			$method,
			$url,
			[],
			$body
		);

		try {
			$response = $client->send($request);
		} catch (ConnectException $e) {
			throw new \Error(
				"Cannot connect to the ociswrapper at the moment,"
				. "make sure that ociswrapper is running before proceeding with the test run.\n"
				. $e->getMessage()
			);
		} catch (GuzzleException $ex) {
			$response = $ex->getResponse();

			if ($response === null) {
				throw $ex;
			}
		}

		return $response;
	}

	/**
	 * @return string
	 */
	public static function getWrapperUrl(): string {
		$url = \getenv("OCIS_WRAPPER_URL");
		if ($url === false) {
			$url = "http://localhost:5200";
		}
		return $url;
	}

	/**
	 * @param array $envs
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function reConfigureOcis(array $envs): ResponseInterface {
		$url = self::getWrapperUrl() . "/config";
		return self::sendRequest($url, "PUT", \json_encode($envs));
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function rollbackOcis(): ResponseInterface {
		$url = self::getWrapperUrl() . "/rollback";
		return self::sendRequest($url, "DELETE");
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function stopOcis(): ResponseInterface {
		$url = self::getWrapperUrl() . "/stop";
		return self::sendRequest($url, "POST");
	}

	/**
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function startOcis(): ResponseInterface {
		$url = self::getWrapperUrl() . "/start";
		return self::sendRequest($url, "POST");
	}

	/**
	 * this method stops the running oCIS instance,
	 * restarts it while excluding specific services,
	 * and then starts the excluded services separately.
	 *
	 * @param string $service
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function startService(string $service): ResponseInterface {
		$envs = [
			"OCIS_LOG_LEVEL" => "info",
		];
		$url = self::getWrapperUrl() . "/services/" . $service;
		return self::sendRequest($url, "POST", \json_encode($envs));
	}
}
