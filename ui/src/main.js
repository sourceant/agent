import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from '~/App.vue'
import Overview from '~/pages/Overview.vue'
import Graph from '~/pages/Graph.vue'
import Knowledge from '~/pages/Knowledge.vue'
import Repositories from '~/pages/Repositories.vue'
import SettingsPage from '~/pages/Settings.vue'
import '~/assets/main.css'

// Hash history, because the agent serves one file and knows nothing about
// paths a router invented.
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: Overview },
    { path: '/graph', component: Graph },
    { path: '/knowledge', component: Knowledge },
    { path: '/repositories', component: Repositories },
    { path: '/settings', component: SettingsPage },
    { path: '/:rest(.*)*', redirect: '/' },
  ],
})

createApp(App).use(router).mount('#app')
