import {defineWebApplication} from '@ownclouders/web-pkg'
import App from './App.vue'

const applicationId = 'hello-app'

export default defineWebApplication({
    setup({applicationConfig}) {
        return {
            appInfo: {
                name: 'Hello App',
                id: 'hello-app',
                icon: 'contrast-2',
                color: '#ff0000',
                isFileEditor: false,
                applicationMenu: {
                    enabled: () => true,
                    priority: 300,
                },
            },
            routes: [
                {
                    path: '/',
                    name: 'hello-app-home',
                    component: App,
                    props: {
                        greetings: ['Hola', 'Bonjour', 'Hallo', 'Nǐ hǎo', 'S̄wạs̄dī', 'Jambo', 'Bonjou', 'Ciao'],
                        ...applicationConfig,
                    },
                    meta: {
                        authContext: 'user',
                        title: 'Home'
                    }
                }
            ]
        }
    }
})