import asyncio
import datetime
import hashlib
import json
import os
import re
from typing import Dict, Any, Union, List
import html

from crawl4ai import AsyncWebCrawler, CrawlerRunConfig, CacheMode
from crawl4ai.extraction_strategy import JsonCssExtractionStrategy
# 使用标准的 NATS 库，支持 JetStream
from nats.aio.client import Client as NATS


def md5(text):
    m = hashlib.md5()
    m.update(text.encode('utf-8'))
    return m.hexdigest()


def get_datas(file_path, json_flag=True, all_flag=False, mode='r'):
    """读取文本文件"""
    results = []

    if not os.path.exists(file_path):
        return results

    try:
        with open(file_path, mode, encoding='utf-8') as f:
            for line in f.readlines():
                if json_flag:
                    try:
                        results.append(json.loads(line.strip()))
                    except json.JSONDecodeError:
                        continue
                else:
                    results.append(line.strip())
        if all_flag:
            if json_flag:
                return json.loads(''.join(results))
            else:
                return '\n'.join(results)
        return results
    except Exception as e:
        print(f"读取文件失败: {e}")
        return results


def save_datas(file_path, datas, json_flag=True, all_flag=False, with_indent=False, mode='w'):
    """保存文本文件"""
    try:
        with open(file_path, mode, encoding='utf-8') as f:
            if all_flag:
                if json_flag:
                    f.write(json.dumps(datas, ensure_ascii=False, indent= 4 if with_indent else None))
                else:
                    f.write(''.join(datas))
            else:
                for data in datas:
                    if json_flag:
                        f.write(json.dumps(data, ensure_ascii=False) + '\n')
                    else:
                        f.write(data + '\n')
    except Exception as e:
        print(f"保存文件失败: {e}")


class AbstractAICrawler():

    def __init__(self) -> None:
        pass
    def crawl(self):
        raise NotImplementedError()


class AINewsCrawler(AbstractAICrawler):
    def __init__(self, domain) -> None:
        super().__init__()
        self.domain = domain
        self.file_path = f'data/{self.domain}.json'
        self.history = self.init()

    def init(self):
        if not os.path.exists(self.file_path):
            return {}
        try:
            data = get_datas(self.file_path)
            return {ele['id']: ele for ele in data if 'id' in ele}
        except Exception as e:
            print(f"初始化历史数据失败: {e}")
            return {}

    def save(self, datas: Union[List, Dict]):
        if isinstance(datas, dict):
            datas = [datas]
        self.history.update({ele['id']: ele for ele in datas})
        save_datas(self.file_path, datas=list(self.history.values()))


class FinanceNewsCrawler(AINewsCrawler):

    def __init__(self, domain='') -> None:
        super().__init__(domain)

    def save(self, datas: Union[List, Dict]):
        if isinstance(datas, dict):
            datas = [datas]
        self.history.update({ele['id']: ele for ele in datas})

        # 确保目录存在
        os.makedirs(os.path.dirname(self.file_path), exist_ok=True)

        # 检查文件是否存在，决定是追加还是新建
        file_exists = os.path.exists(self.file_path)
        mode = 'a' if file_exists else 'w'

        save_datas(self.file_path, datas=datas, mode=mode)
        print(f"数据已保存到 {self.file_path}，模式: {mode}")

    async def get_last_day_data(self):
        last_day = (datetime.date.today() - datetime.timedelta(days=1)).strftime('%Y-%m-%d')
        datas = self.init()
        return [v for v in datas.values() if last_day in v.get('date', '')]


class CLSCrawler(FinanceNewsCrawler):
    """财联社新闻抓取 - 支持SPA应用"""

    def __init__(self) -> None:
        self.domain = 'cls'
        super().__init__(self.domain)
        self.url = 'https://www.cls.cn'
        self.nats_client = None
        self.js = None  # JetStream 上下文

    async def connect_nats(self):
        """连接到NATS服务器并设置JetStream"""
        try:
            self.nats_client = NATS()
            await self.nats_client.connect("nats://localhost:4222")
            print("✅ 已连接到NATS服务器")

            # 获取JetStream上下文
            self.js = self.nats_client.jetstream()

            # 创建Stream（如果不存在）
            try:
                await self.js.add_stream(
                    name="NEWS_STREAM",
                    subjects=["news.>"],  # 匹配所有以news.开头的主题
                    retention="limits",   # 基于限制的保留策略
                    max_msgs=10000,      # 最大消息数
                    max_bytes=100*1024*1024,  # 最大字节数 100MB
                    max_age=7*24*3600    # 保留7天
                )
                print("✅ 创建了JetStream流: NEWS_STREAM")
            except Exception as e:
                if "stream name already in use" in str(e):
                    print("📋 JetStream流 NEWS_STREAM 已存在")
                else:
                    print(f"⚠️ 创建JetStream流时出错: {e}")

        except Exception as e:
            print(f"❌ 连接NATS失败: {e}")
            self.nats_client = None
            self.js = None

    async def publish_news(self, news_data):
        """发布新闻数据到NATS JetStream"""
        if self.nats_client is None or self.js is None:
            await self.connect_nats()

        if self.nats_client is None or self.js is None:
            print("❌ NATS JetStream连接失败，跳过发布")
            return

        try:
            # 发送到news.cls主题
            message = json.dumps(news_data, ensure_ascii=False).encode('utf-8')

            # 使用JetStream发布，会返回ACK
            ack = await self.js.publish("news.cls", message)
            print(f"✅ 已发布新闻到NATS JetStream: {news_data['title']}")
            print(f"📊 消息序号: {ack.seq}, 流: {ack.stream}")

        except Exception as e:
            print(f"❌ 发布新闻失败: {e}")

    async def debug_rendered_page(self, url, crawler):
        """调试已渲染页面的实际DOM结构"""

        js_commands = [
            """
            (async () => {
                console.log('🔍 开始调试页面:', window.location.href);
                
                // 等待页面完全渲染
                await new Promise(resolve => setTimeout(resolve, 10000));
                
                // 等待内容加载
                let attempts = 0;
                while (attempts < 20) {
                    const titleElements = document.querySelectorAll('h1, h2, [class*="title"]');
                    const contentElements = document.querySelectorAll('[class*="content"], [class*="article"], [class*="detail"]');
                    
                    if (titleElements.length > 0 && contentElements.length > 0) {
                        console.log('✅ 内容已加载');
                        break;
                    }
                    
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    attempts++;
                }
                
                // 分析页面结构
                console.log('=== 🔍 页面结构分析 ===');
                
                // 查找所有可能的标题元素
                const titleSelectors = ['h1', 'h2', 'h3', '[class*="title"]', '[class*="headline"]'];
                titleSelectors.forEach(selector => {
                    const elements = document.querySelectorAll(selector);
                    elements.forEach((el, index) => {
                        if (el.textContent.trim().length > 10) {
                            console.log(`📰 标题候选 ${selector}[${index}]:`, el.textContent.trim().substring(0, 50));
                            console.log(`  - 类名:`, el.className);
                            console.log(`  - 长度:`, el.textContent.trim().length);
                        }
                    });
                });
                
                // 查找所有可能的内容元素
                const contentSelectors = ['[class*="content"]', '[class*="article"]', '[class*="detail"]', '[class*="text"]'];
                contentSelectors.forEach(selector => {
                    const elements = document.querySelectorAll(selector);
                    elements.forEach((el, index) => {
                        if (el.textContent.trim().length > 100) {
                            console.log(`📝 内容候选 ${selector}[${index}]:`, el.textContent.trim().substring(0, 100));
                            console.log(`  - 类名:`, el.className);
                            console.log(`  - 长度:`, el.textContent.trim().length);
                        }
                    });
                });
                
                // 查找时间元素
                const timeSelectors = ['[class*="time"]', '[class*="date"]', 'time'];
                timeSelectors.forEach(selector => {
                    const elements = document.querySelectorAll(selector);
                    elements.forEach((el, index) => {
                        console.log(`⏰ 时间候选 ${selector}[${index}]:`, el.textContent.trim());
                        console.log(`  - 类名:`, el.className);
                    });
                });
                
                // 查找作者元素
                const authorSelectors = ['[class*="author"]', '[class*="reporter"]', '[class*="editor"]'];
                authorSelectors.forEach(selector => {
                    const elements = document.querySelectorAll(selector);
                    elements.forEach((el, index) => {
                        console.log(`👤 作者候选 ${selector}[${index}]:`, el.textContent.trim());
                        console.log(`  - 类名:`, el.className);
                    });
                });
                
                return true;
            })();
            """
        ]

        try:
            config = CrawlerRunConfig(
                cache_mode=CacheMode.BYPASS,
                page_timeout=120000,
                delay_before_return_html=15.0,
                js_code=js_commands,
                magic=True,
                remove_overlay_elements=False,
                process_iframes=False,
                exclude_external_links=True,
                wait_until='networkidle'
            )

            result = await crawler.arun(url=url, config=config)

            if result.success:
                print(f"🔍 调试页面: {url}")
                print(f"📄 页面HTML长度: {len(result.html)}")

                # 保存HTML到文件供分析
                debug_file = f"debug_{url.split('/')[-1]}.html"
                with open(debug_file, 'w', encoding='utf-8') as f:
                    f.write(result.html)
                print(f"📁 HTML已保存到: {debug_file}")

                # 用BeautifulSoup分析结构
                from bs4 import BeautifulSoup
                soup = BeautifulSoup(result.html, 'html.parser')

                print("\n🔍 DOM结构分析:")

                # 查找标题
                print("\n📰 标题元素:")
                title_elements = soup.find_all(['h1', 'h2', 'h3']) + soup.find_all(class_=re.compile(r'title|headline'))
                for i, elem in enumerate(title_elements[:5]):
                    text = elem.get_text().strip()
                    if len(text) > 10:
                        print(f"  {i+1}. 标签: {elem.name}, 类: {elem.get('class', [])}")
                        print(f"     文本: {text[:80]}...")

                # 查找内容
                print("\n📝 内容元素:")
                content_elements = soup.find_all(class_=re.compile(r'content|article|detail|text'))
                for i, elem in enumerate(content_elements[:5]):
                    text = elem.get_text().strip()
                    if len(text) > 100:
                        print(f"  {i+1}. 标签: {elem.name}, 类: {elem.get('class', [])}")
                        print(f"     文本长度: {len(text)}")
                        print(f"     文本预览: {text[:100]}...")

                # 查找时间
                print("\n⏰ 时间元素:")
                time_elements = soup.find_all(class_=re.compile(r'time|date')) + soup.find_all('time')
                for i, elem in enumerate(time_elements[:3]):
                    text = elem.get_text().strip()
                    print(f"  {i+1}. 标签: {elem.name}, 类: {elem.get('class', [])}")
                    print(f"     文本: {text}")

                # 基于调试结果，尝试提取真实的CSS选择器
                print("\n🎯 推荐的CSS选择器:")

                # 分析最可能的标题选择器
                best_title_selector = self.analyze_best_title_selector(soup)
                if best_title_selector:
                    print(f"📰 标题选择器: {best_title_selector}")

                # 分析最可能的内容选择器
                best_content_selector = self.analyze_best_content_selector(soup)
                if best_content_selector:
                    print(f"📝 内容选择器: {best_content_selector}")

            return result.html if result.success else ""

        except Exception as e:
            print(f"❌ 调试失败: {e}")
            return ""

    def analyze_best_title_selector(self, soup):
        """分析最佳的标题选择器"""
        candidates = []

        # 检查h1标签
        h1_elements = soup.find_all('h1')
        for elem in h1_elements:
            text = elem.get_text().strip()
            if 10 < len(text) < 200 and not any(skip in text for skip in ['财联社-主流', '关于我们', '网站声明']):
                candidates.append(('h1', elem.get('class', []), len(text), text[:50]))

        # 检查带title类的元素
        title_elements = soup.find_all(class_=re.compile(r'title'))
        for elem in title_elements:
            text = elem.get_text().strip()
            if 10 < len(text) < 200 and not any(skip in text for skip in ['财联社-主流', '关于我们', '网站声明']):
                classes = elem.get('class', [])
                selector = f".{' '.join(classes)}" if classes else elem.name
                candidates.append((selector, classes, len(text), text[:50]))

        if candidates:
            # 选择最合适的（文本长度适中的）
            best = min(candidates, key=lambda x: abs(x[2] - 30))  # 偏好30字符左右的标题
            return best[0]

        return None

    def analyze_best_content_selector(self, soup):
        """分析最佳的内容选择器"""
        candidates = []

        # 检查带content/article/detail类的元素
        content_patterns = ['content', 'article', 'detail']
        for pattern in content_patterns:
            elements = soup.find_all(class_=re.compile(pattern))
            for elem in elements:
                text = elem.get_text().strip()
                if len(text) > 200:  # 内容应该比较长
                    classes = elem.get('class', [])
                    selector = f".{' '.join(classes)}" if classes else elem.name
                    candidates.append((selector, classes, len(text), text[:100]))

        if candidates:
            # 选择文本最长的作为主要内容
            best = max(candidates, key=lambda x: x[2])
            return best[0]

        return None

    async def debug_specific_urls(self):
        """调试特定URL的页面结构"""
        test_urls = [
            "https://www.cls.cn/detail/2040068",  # 特朗普金卡
            "https://www.cls.cn/detail/2040052",  # 特朗普哈佛
        ]

        async with AsyncWebCrawler(verbose=True, headless=False) as crawler:  # 设置headless=False可以看到浏览器
            for url in test_urls:
                print(f"\n{'='*50}")
                print(f"🔍 调试URL: {url}")
                print(f"{'='*50}")

                html = await self.debug_rendered_page(url, crawler)
                if html:
                    print("✅ 调试完成")
                else:
                    print("❌ 调试失败")

                # 等待一下再处理下一个URL
                await asyncio.sleep(5)

    async def crawl_url_list(self, crawler=None):
        # 更新CSS选择器，根据财联社的实际页面结构
        schema = {
            'name': 'caijingwang toutiao page crawler',
            'baseSelector': 'body',
            'fields': [
                {
                    'name': 'all_links',
                    'selector': 'a[href*="/detail/"]',
                    'type': 'nested_list',
                    'fields': [
                        {'name': 'href', 'type': 'attribute', 'attribute':'href'},
                        {'name': 'text', 'type': 'text'}
                    ]
                }
            ]
        }

        # 增强的JS代码，等待SPA完全渲染
        js_commands = [
            """
            (async () => {
                console.log('开始等待财联社页面渲染...');
                await new Promise(resolve => setTimeout(resolve, 3000));
                
                // 等待React应用初始化
                let retries = 0;
                while (!document.querySelector('a[href*="/detail/"]') && retries < 15) {
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    retries++;
                    console.log('等待链接渲染...', retries);
                }
                
                // 滚动加载更多内容
                window.scrollTo(0, document.body.scrollHeight);
                await new Promise(resolve => setTimeout(resolve, 2000));
                
                // 尝试点击加载更多按钮
                const loadMoreButton = document.querySelector('.more-button, .load-more, [class*="more"]');
                if (loadMoreButton && loadMoreButton.offsetParent !== null) {
                    loadMoreButton.click();
                    await new Promise(resolve => setTimeout(resolve, 3000));
                }
                
                console.log('财联社页面渲染完成');
                return true;
            })();
            """
        ]

        results = {}

        menu_dict = {
            '1000': '头条',
            '1003': 'A股',
            '1007': '环球'
        }

        if crawler is None:
            print("警告：没有传入crawler实例")
            return {}

        for k, v in menu_dict.items():
            url = f'https://www.cls.cn/depth?id={k}'
            print(f"正在抓取 {v} 页面: {url}")
            links = []

            # 重试逻辑
            max_retries = 3
            for retry in range(max_retries):
                try:
                    config = CrawlerRunConfig(
                        extraction_strategy=JsonCssExtractionStrategy(schema, verbose=False),
                        cache_mode=CacheMode.BYPASS,
                        page_timeout=60000,  # 增加超时时间
                        delay_before_return_html=8.0,  # 等待SPA渲染
                        js_code=js_commands,
                        magic=True,
                        remove_overlay_elements=False,  # 不移除元素，避免影响SPA
                        process_iframes=False,
                        exclude_external_links=True,
                        js_only=False,
                        screenshot=False,
                        wait_until='networkidle'
                    )

                    result = await crawler.arun(url=url, config=config)

                    if not result.success:
                        print(f"抓取失败: {url}")
                        continue

                    extracted_data = json.loads(result.extracted_content)
                    print(f"提取的数据结构: {list(extracted_data[0].keys()) if extracted_data else 'empty'}")

                    # 处理提取的链接
                    if extracted_data and 'all_links' in extracted_data[0]:
                        all_links = extracted_data[0]['all_links']
                        news_links = []
                        for link_obj in all_links:
                            if 'href' in link_obj:
                                href = link_obj['href']
                                if '/detail/' in href and not href.startswith('http'):
                                    news_links.append(href)

                        links = list(set(news_links))  # 去重
                        print(f"找到 {len(links)} 个新闻链接")

                    break  # 成功则跳出重试循环

                except Exception as e:
                    print(f'error {url} (尝试 {retry+1}/{max_retries}): {str(e)}')
                    if retry == max_retries - 1:
                        print(f'放弃抓取 {url}')
                    else:
                        await asyncio.sleep(3)

            if links:
                for link in links:
                    full_url = f'{self.url}{link}' if not link.startswith('http') else link
                    results[full_url] = v

        print(f"总共找到 {len(results)} 个新闻链接")
        return results

    async def crawl_newsletter(self, url, category, crawler):
        """针对财联社SPA应用的新闻详情抓取方法"""

        # 增强的JavaScript执行策略
        js_commands = [
            """
            (async () => {
                console.log('开始等待新闻页面渲染...', window.location.href);
                
                // 等待初始渲染
                await new Promise(resolve => setTimeout(resolve, 5000));
                
                // 等待Next.js数据加载
                let dataRetries = 0;
                while (!window.__NEXT_DATA__ && dataRetries < 15) {
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    dataRetries++;
                    console.log('等待Next.js数据...', dataRetries);
                }
                
                // 等待文章内容渲染
                let contentRetries = 0;
                while (!document.querySelector('.detail-content, .article-content, [class*="content"]') && contentRetries < 15) {
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    contentRetries++;
                    console.log('等待内容渲染...', contentRetries);
                }
                
                console.log('新闻页面渲染完成');
                
                // 尝试滚动确保所有内容加载
                window.scrollTo(0, document.body.scrollHeight);
                await new Promise(resolve => setTimeout(resolve, 2000));
                
                return true;
            })();
            """
        ]

        try:
            config = CrawlerRunConfig(
                cache_mode=CacheMode.BYPASS,
                page_timeout=90000,  # 增加超时时间到90秒
                delay_before_return_html=12.0,  # 等待12秒让SPA完全渲染
                js_code=js_commands,
                magic=True,
                remove_overlay_elements=False,  # 不移除元素，避免影响SPA渲染
                process_iframes=False,
                exclude_external_links=True,
                wait_until='networkidle'
            )

            result = await crawler.arun(url=url, config=config)

            if not result.success:
                print(f'❌ 抓取失败: {url}')
                return {}

            # 方法1：从Next.js的JSON数据中提取
            title, publish_time, author, content = self.extract_from_nextjs_data(result.html, url)

            # 方法2：如果JSON提取失败，尝试从渲染后的HTML提取
            if not content or len(content) < 50:
                print("🔄 JSON提取失败，尝试HTML提取...")
                backup_title, backup_content = self.extract_from_rendered_html(result.html)
                if backup_title and not title:
                    title = backup_title
                if backup_content and len(backup_content) > len(content):
                    content = backup_content

            # 方法3：如果还是没有内容，从全文本提取
            if not content or len(content) < 30:
                print("🔄 HTML提取失败，尝试全文本提取...")
                title = self.extract_title_from_all_text(result.html, url)
                publish_time = self.extract_time_from_all_text(result.html)
                author = self.extract_author_from_all_text(result.html)
                content = self.extract_content_from_all_text(result.html)

            print(f"✅ 提取结果:")
            print(f"   📰 标题: {title[:80]}{'...' if len(title) > 80 else ''}")
            print(f"   ⏰ 时间: {publish_time}")
            print(f"   👤 作者: {author}")
            print(f"   📝 内容长度: {len(content)}字符")

            if len(content) < 50:
                print(f"⚠️ 内容过短，可能需要调试")
                print(f"🔍 HTML长度: {len(result.html)}")

        except Exception as e:
            print(f'❌ 爬虫错误: {url} - {str(e)}')
            import traceback
            traceback.print_exc()
            return {}

        return {
            'title': title if title else f"财联社新闻_{url.split('/')[-1]}",
            'abstract': content[:200] + '...' if len(content) > 200 else content,
            'date': publish_time if publish_time else datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'link': url,
            'content': content,
            'id': md5(url),
            'type': category,
            'author': author if author else "财联社",
            'read_number': 0,
            'time': datetime.datetime.now().strftime('%Y-%m-%d')
        }

    def extract_from_nextjs_data(self, html_content, url):
        """从Next.js的JSON数据中提取新闻信息"""
        # 查找 __NEXT_DATA__ 脚本
        next_data_pattern = r'<script id="__NEXT_DATA__" type="application/json">(.*?)</script>'
        match = re.search(next_data_pattern, html_content, re.DOTALL)

        if match:
            try:
                next_data = json.loads(match.group(1))

                # 从JSON数据中提取新闻信息
                article_detail = (next_data.get('props', {})
                                  .get('initialState', {})
                                  .get('detail', {})
                                  .get('articleDetail', {}))

                title = article_detail.get('title', '')
                content = article_detail.get('content', '')
                brief = article_detail.get('brief', '')
                author_info = article_detail.get('author', {})
                author = author_info.get('name', '财联社') if isinstance(author_info, dict) else str(author_info)
                ctime = article_detail.get('ctime', 0)

                # 清理HTML标签
                if content:
                    content = self.clean_html_content(content)

                # 如果没有content，使用brief
                if not content and brief:
                    content = brief

                # 时间转换
                if ctime and isinstance(ctime, (int, float)) and ctime > 0:
                    try:
                        # ctime可能是秒级时间戳
                        if ctime > 1e10:  # 毫秒级时间戳
                            ctime = ctime / 1000
                        publish_time = datetime.datetime.fromtimestamp(ctime).strftime('%Y-%m-%d %H:%M:%S')
                    except (ValueError, OSError):
                        publish_time = datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                else:
                    publish_time = datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

                print(f"📋 从JSON数据提取成功: 标题长度={len(title)}, 内容长度={len(content)}")
                return title, publish_time, author, content

            except json.JSONDecodeError as e:
                print(f"❌ JSON解析失败: {e}")
            except Exception as e:
                print(f"❌ 数据提取失败: {e}")

        print("❌ 未找到Next.js数据")
        return "", "", "财联社", ""

    def clean_html_content(self, content):
        """清理HTML内容，保留纯文本"""
        if not content:
            return ""

        # 移除HTML标签
        content = re.sub(r'<[^>]+>', '', content)

        # 解码HTML实体
        content = html.unescape(content)

        # 清理多余的空白字符
        content = re.sub(r'\s+', ' ', content).strip()

        # 移除特殊字符
        content = re.sub(r'[\u200b\u00a0]', ' ', content)

        return content

    def extract_from_rendered_html(self, html_content):
        """备用方法：从渲染后的HTML中提取信息"""
        try:
            # 使用正则表达式提取标题
            title = ""
            title_patterns = [
                r'<span[^>]*class="[^"]*detail-title-content[^"]*"[^>]*>(.*?)</span>',
                r'<h1[^>]*class="[^"]*title[^"]*"[^>]*>(.*?)</h1>',
                r'<h1[^>]*>(.*?)</h1>',
                r'<title>(.*?)</title>'
            ]

            for pattern in title_patterns:
                match = re.search(pattern, html_content, re.IGNORECASE | re.DOTALL)
                if match:
                    title = self.clean_html_content(match.group(1))
                    if title and len(title) > 5:
                        break

            # 使用正则表达式提取内容
            content = ""
            content_patterns = [
                r'<div[^>]*class="[^"]*detail-content[^"]*"[^>]*>(.*?)</div>',
                r'<div[^>]*class="[^"]*article-content[^"]*"[^>]*>(.*?)</div>',
                r'<div[^>]*class="[^"]*content[^"]*"[^>]*>(.*?)</div>'
            ]

            for pattern in content_patterns:
                match = re.search(pattern, html_content, re.IGNORECASE | re.DOTALL)
                if match:
                    content = self.clean_html_content(match.group(1))
                    if content and len(content) > 50:
                        break

            print(f"📋 从HTML提取: 标题长度={len(title)}, 内容长度={len(content)}")
            return title, content

        except Exception as e:
            print(f"❌ HTML解析失败: {e}")
            return "", ""

    def extract_title_from_all_text(self, text, url):
        """从全文中提取标题"""
        # 移除HTML标签后再处理
        text = re.sub(r'<[^>]+>', ' ', text)
        lines = text.split('\n')

        for line in lines[:50]:  # 看前50行
            line = line.strip()

            if (8 <= len(line) <= 150 and
                    not line.startswith(('关于我们', '网站声明', '联系方式', '用户反馈',
                                         '财联社-主流财经新闻集团', 'http', 'www')) and
                    not line.endswith(('.com', '.cn')) and
                    '©' not in line and
                    '版权' not in line and
                    line.count('|') < 2 and
                    any(char in line for char in ['、', '：', '，', '！', '？', '"', '"']) and
                    not line.isdigit() and
                    not re.match(r'^\d+[-年月日时分秒\s:：]+', line)):

                # 清理标题
                line = re.sub(r'^[^\u4e00-\u9fff]*', '', line)  # 移除开头的非中文字符
                line = re.sub(r'[^\u4e00-\u9fff]*$', '', line)  # 移除结尾的非中文字符

                if len(line) >= 8:
                    return line

        # 从URL提取
        if '/detail/' in url:
            detail_id = url.split('/detail/')[-1]
            return f"财联社新闻_{detail_id}"

        return f"财联社新闻_{datetime.datetime.now().strftime('%Y%m%d_%H%M%S')}"

    def extract_time_from_all_text(self, text):
        """从全文中提取时间"""
        time_patterns = [
            r'(\d{4}年\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2})',
            r'(\d{4}-\d{1,2}-\d{1,2}\s+\d{1,2}:\d{2})',
            r'(\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2})',
            r'(\d{4}年\d{1,2}月\d{1,2}日)',
            r'2025-05-\d{1,2}\s+\d{1,2}:\d{2}',
        ]

        for pattern in time_patterns:
            match = re.search(pattern, text)
            if match:
                return match.group(1) if match.groups() else match.group(0)

        return datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    def extract_author_from_all_text(self, text):
        """从全文中提取作者"""
        author_patterns = [
            r'财联社记者\s+([^\s\n<>]+)',
            r'财联社\s+([^\s\n<>]+)',
            r'记者\s+([^\s\n<>]+)',
            r'编辑\s+([^\s\n<>]+)',
        ]

        for pattern in author_patterns:
            match = re.search(pattern, text)
            if match:
                author = match.group(1).strip()
                if author and len(author) < 10:
                    return author

        return "财联社"

    def extract_content_from_all_text(self, text):
        """从全文中提取正文内容"""
        # 移除HTML标签
        text = re.sub(r'<[^>]+>', '\n', text)
        lines = text.split('\n')
        content_lines = []
        content_started = False

        for line in lines:
            line = line.strip()

            # 跳过导航和页脚
            if (line.startswith(('关于我们', '网站声明', '联系方式', '用户反馈',
                                 '帮助', '首页', '电报', '话题', '盯盘', 'VIP')) or
                    '上证指数' in line or '深证成指' in line or
                    '©' in line or '版权所有' in line or
                    len(line) < 15):
                continue

            # 寻找正文开始
            if ('财联社' in line and ('日讯' in line or '电' in line)) or \
                    (len(line) > 30 and ('。' in line or '，' in line) and
                     not line.startswith(('http', 'www'))):
                content_started = True

            # 收集正文
            if (content_started and len(line) > 15 and
                    not line.startswith(('收藏', '阅读', '我要评论', '反馈意见',
                                         '要闻', '股市', '查看更多', '关联话题')) and
                    not line.endswith(('.com', '.cn'))):
                content_lines.append(line)

            if len(content_lines) > 20:
                break

        result = '\n'.join(content_lines[:15])
        return result if len(result) > 30 else ""

    async def crawl(self, debug_mode=False):
        """主爬取方法"""
        # 如果是调试模式，只运行调试功能
        if debug_mode:
            print("🔍 进入调试模式...")
            await self.debug_specific_urls()
            return []

        # 确保数据目录存在
        os.makedirs('data', exist_ok=True)

        # 连接NATS JetStream
        try:
            await self.connect_nats()
        except Exception as e:
            print(f"NATS JetStream连接失败，将跳过消息发布: {e}")

        # 创建浏览器实例
        async with AsyncWebCrawler(verbose=False,
                                   headless=True,
                                   always_by_pass_cache=True) as crawler:
            print("开始获取新闻链接...")
            link_2_category = await self.crawl_url_list(crawler=crawler)

            if not link_2_category:
                print("未获取到任何新闻链接，退出...")
                return []

            print(f"获取到 {len(link_2_category)} 个新闻链接，开始抓取详情...")

            processed_count = 0
            for link, category in link_2_category.items():
                _id = md5(link)
                if _id in self.history:
                    print(f"跳过已存在的新闻: {link}")
                    continue

                print(f"正在处理第 {processed_count + 1} 个新闻: {link}")

                try:
                    news = await self.crawl_newsletter(link, category, crawler)

                    if not news or not news.get('title') or len(news.get('title', '').strip()) < 5:
                        print(f"新闻内容提取失败，跳过: {link}")
                        continue

                    print(f"保存新闻: {news['title']}")
                    self.save(news)

                    # 发布到NATS JetStream
                    await self.publish_news(news)

                    processed_count += 1

                    # 每处理3个新闻休息一下
                    if processed_count % 3 == 0:
                        print(f"已处理 {processed_count} 个新闻，暂停3秒...")
                        await asyncio.sleep(3)

                except Exception as e:
                    print(f'处理新闻详情失败: {link} - {str(e)}')
                    continue

        # 关闭NATS连接
        if self.nats_client:
            try:
                await self.nats_client.close()
                print("🔚 NATS JetStream连接已关闭")
            except Exception as e:
                print(f"❌ 关闭NATS JetStream连接失败: {e}")

        print(f"爬取完成，共处理 {processed_count} 个新闻")
        return await self.get_last_day_data()


async def test_nats_jetstream_connection():
    """测试NATS JetStream连接"""
    try:
        nc = NATS()
        await nc.connect("nats://localhost:4222")
        print("✅ NATS连接测试成功")

        js = nc.jetstream()

        # 尝试发布一条测试消息
        test_msg = {"test": "message", "timestamp": datetime.datetime.now().isoformat()}
        ack = await js.publish("test.subject", json.dumps(test_msg).encode())
        print(f"✅ JetStream发布测试成功，序号: {ack.seq}")

        await nc.close()
        return True
    except Exception as e:
        print(f"❌ NATS JetStream连接测试失败: {e}")
        return False


if __name__ == '__main__':
    import sys

    # 检查命令行参数
    debug_mode = '--debug' in sys.argv

    if debug_mode:
        print("🔍 启动调试模式...")
        crawler = CLSCrawler()
        asyncio.run(crawler.crawl(debug_mode=True))
    else:
        # 先测试NATS JetStream连接
        asyncio.run(test_nats_jetstream_connection())

        # 然后运行爬虫
        asyncio.run(CLSCrawler().crawl())
