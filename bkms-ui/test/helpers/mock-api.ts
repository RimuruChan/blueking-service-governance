/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */
/**
 * 「壳层契约测试」共享 API mock 工厂（§3.7 禁止重复造轮子）。
 *
 * 适用：需要 mount 真实页面/壳层组件、但不验证 API 行为的契约类测试
 * （height-chain / app-detail-shell）。目的是掐断 API 加载链让组件可 import/mount，
 * 测试路径不触达返回值——需要断言 API 调用的场景请用 vi.hoisted + 显式 vi.fn。
 */

/**
 * bkmsserver 最小集：height-chain / app-detail-shell 挂载时触达的显式方法。
 * GetApmServiceName 显式 stub：observation.vue fetchApmData 靠前置 guard 短路，
 * 此处兜底避免未来 guard 调整后以 TypeError 形式挂掉。
 */
import { vi } from 'vitest';

export function bkmsserverShellMockFactory() {
  return {
    ApiServerService: {
      ListApps: vi.fn().mockResolvedValue([]),
      GetWorkspace: vi.fn().mockResolvedValue(undefined),
      GetApp: vi.fn().mockResolvedValue(undefined),
      GetApmServiceName: vi.fn().mockResolvedValue(null),
    },
  };
}

/**
 * v1 API 模块自相似 Proxy 兜底：v1 导出大量 service 且部分子组件在顶层 .bind() 引用，
 * 任意导出名均得到可调用、可 bind 的节点（测试路径不触达其返回值）。
 * 注意：symbol/then/__esModule 必须透传 undefined，否则 vi.mock 的 await 会把 factory
 * 结果误判为 thenable 导致 vitest 挂死；模块层须为 object（vitest 校验 factory 返回值）。
 */
export function v1ProxyMockFactory() {
  const makeNode = (): unknown =>
    new Proxy(function stub() {} as unknown as Record<PropertyKey, unknown>, {
      get: (target, prop) => {
        if (typeof prop === 'symbol' || prop === 'then' || prop === '__esModule' || prop === 'default') {
          return undefined;
        }
        if (!(prop in target)) {
          target[prop] = makeNode();
        }
        return target[prop];
      },
      apply: () => Promise.resolve(undefined),
    });
  const moduleCache: Record<PropertyKey, unknown> = {};
  return new Proxy(moduleCache, {
    get: (target, prop) => {
      if (typeof prop === 'symbol' || prop === 'then' || prop === '__esModule') {
        return undefined;
      }
      if (!(prop in target)) {
        target[prop] = makeNode();
      }
      return target[prop];
    },
  });
}
