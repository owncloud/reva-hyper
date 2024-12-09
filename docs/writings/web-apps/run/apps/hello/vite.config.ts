import {defineConfig, loadEnv} from 'vite'
import {defineConfig as defineWebConfig} from '@ownclouders/extension-sdk'
import generateFile from 'vite-plugin-generate-file'

export default defineConfig((args) => {
    const env = loadEnv(args.mode, process.cwd(), '')

    return defineWebConfig({
        build: {
            rollupOptions: {
                output: {
                    entryFileNames: 'index.js'
                }
            }
        },
        plugins: [
            generateFile([{
                type: 'json',
                output: 'manifest.json',
                data: {
                    "id": "hello-app",
                    "entrypoint": "index.js",
                    "config": {
                        "salutation": env.APP_HELLO_SALUTION || "REPLACE VIA ENV!!!!!",
                    }
                }
            }])
        ],
    })(args)
})