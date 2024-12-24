<?php
namespace TestHelpers;

use http\Env\Response;
use TusPhp\File;
use Carbon\Carbon;
use TusPhp\Config;
use Ramsey\Uuid\Uuid;
use TusPhp\Exception\TusException;
use TusPhp\Exception\FileException;
use GuzzleHttp\ClientTrait;
use GuzzleHttp\Client as GuzzleClient;
use GuzzleHttp\Exception\ClientException;
use TusPhp\Exception\ConnectionException;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Exception\ConnectException;
use Symfony\Component\HttpFoundation\Response as HttpResponse;
use ReflectionException;
use TusPhp\Tus\Client;

class TusClientWrapper extends Client{
    public $client;

    /**
     * Constructor for the TusClientWrapper.
     *
     * @param string $baseUri Base URI of the TUS server.
     * @param array $options Additional client options.
     * @throws ReflectionException
     */
    public function __construct(string $baseUri, array $options = []) {
        parent::__construct($baseUri,$options);
        $this->client = new Client($baseUri,$options);
    }

    public function setKey(string $key): self
    {
        $this->client->setKey($key);
        return $this;
    }

    public function file(string $file, string $name = null): self
    {
        $this->client->file($file,$name);
        return $this;
    }

    public function createWithUpload(string $key, int $bytes = -1): array {
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
            $response = $this->client->getClient()->post($this->apiPath, [
                'body' => $data,
                'headers' => $headers,
            ]);
        } catch (ClientException $e) {
            $response = $e->getResponse();
        }

        $statusCode = $response->getStatusCode();
        $header = $response->getHeaders();


        if (HttpResponse::HTTP_CREATED !== $statusCode) {
            throw new FileException('Unable to create resource.');
        }

        $uploadOffset   = $bytes > 0 ? current($response->getHeader('upload-offset')) : 0;
        $uploadLocation = current($response->getHeader('location'));

        $this->getCache()->set($this->client->getKey(), [
            'location' => $uploadLocation,
            'expires_at' => Carbon::now()->addSeconds($this->getCache()->getTtl())->format($this->getCache()::RFC_7231),
        ]);

        return [
            'location' => $uploadLocation,
            'offset' => $uploadOffset,
        ];
    }

//    /**
//     * Set API path.
//     *
//     * @param string $path
//     *
//     * @return self
//     */
//    public function setApiPath(string $path): self
//    {
//        $this->client->apiPath = $path;
//        return  $this;
//    }

//    public function setApiPath(string $path): self
//    {
//        $this->client->apiPath = $path;
//        return  $this;
//    }

//    public function setKey(string $key): self
//    {
//        $this->client->setKey($key);
//        return $this;
//    }

//    public function file(string $file, string $name = null): self
//    {
//        $this->client->file($file,$name);
//        return $this;
//    }

//    public function createWithUpload(string $key, int $bytes = -1): void
//    {
//        $this->client->createWithUpload($key,$bytes);
//    }

//    public function setHeaders(array $headers): void {
//        $this->client->setHeaders($headers);
//    }
//
//    public function upload(string $filePath): string {
//        $this->client->setFilePath($filePath);
//        return $this->client->upload();
//    }
//
//    public function getOffset(): int {
//        return $this->client->getOffset();
//    }
//
//    public function setChunkSize(int $bytes): void  {
//        $this->client->setChunkSize($bytes);
//    }
//
//    public function terminateUpload(string $uploadUrl): void  {
//        $this->client->terminate($uploadUrl);
//    }
//
//    public function setKey(string $key): self
//    {
//        $this->client->key = $key;
//
//        return $this;
//    }
//
//    public function file(string $file, string $name = null): self
//    {
//        $this->filePath = $file;
//
//        if ( ! file_exists($file) || ! is_readable($file)) {
//            throw new FileException('Cannot read file: ' . $file);
//        }
//
//        $this->fileName = $name ?? basename($this->filePath);
//        $this->fileSize = filesize($file);
//
//        $this->addMetadata('filename', $this->fileName);
//
//        return $this;
//    }

}
