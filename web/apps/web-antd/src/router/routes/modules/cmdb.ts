import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'lucide:database',
      order: 10,
      title: 'CMDB',
    },
    name: 'Cmdb',
    path: '/cmdb',
    children: [
      {
        name: 'CmdbProject',
        path: '/cmdb/project',
        component: () => import('#/views/cmdb/project/index.vue'),
        meta: {
          icon: 'lucide:folder-kanban',
          title: '项目管理',
        },
      },
      {
        name: 'CmdbApplication',
        path: '/cmdb/application',
        component: () => import('#/views/cmdb/application/index.vue'),
        meta: {
          icon: 'lucide:app-window',
          title: '应用管理',
        },
      },
      {
        name: 'CmdbHost',
        path: '/cmdb/host',
        component: () => import('#/views/cmdb/host/index.vue'),
        meta: {
          icon: 'lucide:server',
          title: '主机管理',
        },
      },
      {
        meta: {
          icon: 'lucide:boxes',
          title: '中间件',
        },
        name: 'CmdbMiddleware',
        path: '/cmdb/middleware',
        children: [
          {
            name: 'CmdbMiddlewareMySQL',
            path: '/cmdb/middleware/mysql',
            component: () => import('#/views/cmdb/middleware/mysql/index.vue'),
            meta: {
              icon: 'lucide:database',
              title: 'MySQL',
            },
          },
          {
            name: 'CmdbMiddlewareRedis',
            path: '/cmdb/middleware/redis',
            component: () => import('#/views/cmdb/middleware/redis/index.vue'),
            meta: {
              icon: 'lucide:database',
              title: 'Redis',
            },
          },
          {
            name: 'CmdbMiddlewareNginx',
            path: '/cmdb/middleware/nginx',
            component: () => import('#/views/cmdb/middleware/nginx/index.vue'),
            meta: {
              icon: 'lucide:globe',
              title: 'Nginx',
            },
          },
          {
            name: 'CmdbMiddlewareTomcat',
            path: '/cmdb/middleware/tomcat',
            component: () => import('#/views/cmdb/middleware/tomcat/index.vue'),
            meta: {
              icon: 'lucide:coffee',
              title: 'Tomcat',
            },
          },
          {
            name: 'CmdbMiddlewareElasticsearch',
            path: '/cmdb/middleware/elasticsearch',
            component: () => import('#/views/cmdb/middleware/elasticsearch/index.vue'),
            meta: {
              icon: 'lucide:search',
              title: 'Elasticsearch',
            },
          },
        ],
      },
      {
        name: 'CmdbInspection',
        path: '/cmdb/inspection',
        component: () => import('#/views/cmdb/inspection/index.vue'),
        meta: {
          icon: 'lucide:clipboard-check',
          title: '巡检管理',
        },
      },
      {
        name: 'CmdbAlert',
        path: '/cmdb/alert',
        component: () => import('#/views/cmdb/alert/index.vue'),
        meta: {
          icon: 'lucide:bell',
          title: '告警列表',
        },
      },
    ],
  },
];

export default routes;
