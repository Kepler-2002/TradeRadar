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
from bs4 import BeautifulSoup


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

    def create_enhanced_crawler_config(self, headless=True):
        """创建增强的爬虫配置"""
        return {
            'headless': headless,
            'verbose': False,  # 减少日志输出
            'browser_type': 'chromium',
            'user_agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            'headers': {
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
                'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
                'Accept-Encoding': 'gzip, deflate, br',
                'Connection': 'keep-alive',
                'Upgrade-Insecure-Requests': '1',
                'Sec-Fetch-Dest': 'document',
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'none',
                'Cache-Control': 'max-age=0'
            },
            'extra_args': [
                '--disable-blink-features=AutomationControlled',
                '--disable-web-security',
                '--disable-features=VizDisplayCompositor',
                '--no-sandbox',
                '--disable-dev-shm-usage',
                '--disable-gpu',
                '--window-size=1920,1080',
                '--disable-extensions',
                '--disable-plugins',
                '--disable-images',  # 禁用图片加载以提高速度
                '--disable-javascript',  # 暂时禁用JS，避免复杂的SPA渲染
            ],
            'sleep_on_close': False
        }

    async def simple_crawl_with_retry(self, url, crawler, max_retries=3):
        """简单的爬取方法，带重试机制"""
        for attempt in range(max_retries):
            try:
                print(f"🌐 尝试访问 ({attempt + 1}/{max_retries}): {url}")

                # 简化的配置，避免复杂的等待条件
                config = CrawlerRunConfig(
                    cache_mode=CacheMode.BYPASS,
                    page_timeout=30000,  # 30秒超时
                    delay_before_return_html=3.0,  # 简化等待时间
                    magic=False,  # 禁用magic模式
                    remove_overlay_elements=False,
                    exclude_external_links=True,
                    # 移除wait_until参数，避免networkidle超时
                )

                result = await crawler.arun(url=url, config=config)

                if result.success and result.html:
                    print(f"✅ 访问成功: {url}")
                    return result
                else:
                    print(f"❌ 访问失败 (尝试 {attempt + 1}): {result.error_message if result else 'Unknown error'}")

            except Exception as e:
                print(f"❌ 访问异常 (尝试 {attempt + 1}): {e}")

            if attempt < max_retries - 1:
                wait_time = (attempt + 1) * 2  # 递增等待时间
                print(f"⏰ 等待 {wait_time} 秒后重试...")
                await asyncio.sleep(wait_time)

        print(f"❌ 所有尝试都失败了: {url}")
        return None

    async def debug_specific_urls(self):
        """调试特定URL的页面结构"""
        test_urls = [
            "https://www.cls.cn/detail/2040068",  # 特朗普金卡
            "https://www.cls.cn/detail/2040052",  # 特朗普哈佛
        ]

        # 使用更保守的配置
        crawler_config = self.create_enhanced_crawler_config(headless=True)

        async with AsyncWebCrawler(**crawler_config) as crawler:
            for url in test_urls:
                print(f"\n{'='*50}")
                print(f"🔍 调试URL: {url}")
                print(f"{'='*50}")

                # 直接访问目标页面，不再先访问首页
                result = await self.simple_crawl_with_retry(url, crawler)

                if result and result.success:
                    print(f"📄 页面HTML长度: {len(result.html)}")

                    # 保存HTML到文件供分析
                    debug_file = f"debug_{url.split('/')[-1]}.html"
                    with open(debug_file, 'w', encoding='utf-8') as f:
                        f.write(result.html)
                    print(f"📁 HTML已保存到: {debug_file}")

                    # 分析页面内容
                    article_data = self.extract_article_content_advanced(result.html, url)

                    print(f"\n🔍 文章提取结果:")
                    print(f"📰 标题: {article_data.get('title', 'N/A')[:100]}...")
                    print(f"📝 内容长度: {len(article_data.get('content', ''))}")
                    print(f"📝 内容预览: {article_data.get('content', '')[:200]}...")
                    print(f"⏰ 时间: {article_data.get('time', 'N/A')}")
                    print(f"👤 作者: {article_data.get('author', 'N/A')}")
                    print(f"✅ 提取成功: {article_data.get('success', False)}")

                    # 检查页面是否被重定向
                    if self.is_redirected_to_homepage(result.html):
                        print("\n⚠️  警告: 页面被重定向到首页!")
                        print("可能的原因:")
                        print("1. 网站反爬虫机制")
                        print("2. 需要特殊的访问权限")
                        print("3. URL无效或文章已删除")

                        # 尝试其他方法
                        print("\n🔄 尝试备用访问方法...")
                        await self.try_alternative_access(url, crawler)
                    else:
                        print("✅ 页面内容正常")

                else:
                    print(f"❌ 调试失败: {url}")

                # 等待一下再处理下一个URL
                await asyncio.sleep(3)

        print("✅ 调试完成")

    def is_redirected_to_homepage(self, html):
        """检查是否被重定向到首页"""
        indicators = [
            "财联社-主流财经新闻集团",
            "上证指数",
            "深证成指",
            "创业板指",
            "看盘",
            "热门板块"
        ]

        # 简单检查HTML内容
        html_lower = html.lower()

        # 如果包含太多首页特征，认为是重定向
        homepage_count = sum(1 for indicator in indicators if indicator in html)

        return homepage_count >= 3

    async def try_alternative_access(self, url, crawler):
        """尝试备用访问方法"""
        alternatives = [
            # 方法1: 添加随机参数
            f"{url}?t={int(datetime.datetime.now().timestamp())}",
            # 方法2: 尝试移动版
            url.replace('www.cls.cn', 'm.cls.cn'),
        ]

        for i, alt_url in enumerate(alternatives):
            print(f"🔄 尝试方法 {i+1}: {alt_url}")

            try:
                result = await self.simple_crawl_with_retry(alt_url, crawler, max_retries=2)

                if result and result.success and not self.is_redirected_to_homepage(result.html):
                    print(f"✅ 备用方法 {i+1} 成功!")
                    # 保存成功的HTML
                    success_file = f"success_{url.split('/')[-1]}_method{i+1}.html"
                    with open(success_file, 'w', encoding='utf-8') as f:
                        f.write(result.html)
                    print(f"📁 成功的HTML已保存到: {success_file}")
                    return result
                else:
                    print(f"❌ 备用方法 {i+1} 失败")

            except Exception as e:
                print(f"❌ 备用方法 {i+1} 出错: {e}")

            await asyncio.sleep(2)

        return None

    def extract_article_content_advanced(self, html_content, url):
        """高级内容提取方法"""
        soup = BeautifulSoup(html_content, 'html.parser')

        # 检查是否是首页
        if self.is_redirected_to_homepage(html_content):
            print("⚠️  检测到首页重定向，尝试从HTML中提取可能的信息")
            return self.extract_from_homepage_html(html_content, url)

        # 方法1: 从JSON数据提取
        title, content, time_str, author = self.extract_from_script_json(html_content)

        # 方法2: 从DOM元素提取
        if not content or len(content) < 50:
            title2, content2, time_str2, author2 = self.extract_from_dom_elements(soup)
            if len(content2) > len(content):
                title, content, time_str, author = title2, content2, time_str2, author2

        # 方法3: 智能文本提取
        if not content or len(content) < 30:
            title3, content3, time_str3, author3 = self.extract_from_smart_text(html_content)
            if len(content3) > len(content):
                title, content, time_str, author = title3, content3, time_str3, author3

        return {
            'title': title or f"财联社新闻_{url.split('/')[-1]}",
            'content': content or "",
            'time': time_str or datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'author': author or "财联社",
            'success': bool(title and content and len(content) > 30)
        }

    def extract_from_homepage_html(self, html_content, url):
        """从首页HTML中尝试提取相关新闻信息"""
        # 从URL提取文章ID
        article_id = url.split('/')[-1] if '/' in url else ""

        # 在首页HTML中搜索相关的新闻片段
        soup = BeautifulSoup(html_content, 'html.parser')

        # 查找可能包含目标文章信息的元素
        links = soup.find_all('a', href=re.compile(f'/detail/{article_id}'))

        title = ""
        content = ""

        for link in links:
            # 获取链接文本作为标题
            link_text = link.get_text().strip()
            if link_text and len(link_text) > 8:
                title = link_text

                # 查找相邻的内容元素
                parent = link.parent
                if parent:
                    siblings = parent.find_next_siblings()
                    for sibling in siblings[:3]:  # 检查后面几个兄弟元素
                        sibling_text = sibling.get_text().strip()
                        if len(sibling_text) > 50:
                            content = sibling_text
                            break
                break

        # 如果还是没找到，使用文章ID作为标题
        if not title:
            title = f"财联社新闻_{article_id}"

        return {
            'title': title,
            'content': content,
            'time': datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            'author': "财联社",
            'success': bool(title)
        }

    def extract_from_script_json(self, html_content):
        """从页面JSON数据中提取"""
        # 查找 __NEXT_DATA__
        next_data_pattern = r'<script id="__NEXT_DATA__" type="application/json">(.*?)</script>'
        match = re.search(next_data_pattern, html_content, re.DOTALL)

        if match:
            try:
                next_data = json.loads(match.group(1))

                # 递归搜索文章数据
                article_data = self.find_article_data_in_json(next_data)

                if article_data:
                    title = article_data.get('title', '')
                    content = article_data.get('content', '') or article_data.get('brief', '')

                    # 清理HTML标签
                    if content:
                        content = self.clean_html_content(content)

                    # 处理时间
                    ctime = article_data.get('ctime', 0)
                    time_str = self.convert_timestamp(ctime)

                    # 处理作者
                    author_info = article_data.get('author', {})
                    if isinstance(author_info, dict):
                        author = author_info.get('name', '财联社')
                    else:
                        author = str(author_info) if author_info else '财联社'

                    print(f"📋 JSON提取成功: 标题={len(title)}字, 内容={len(content)}字")
                    return title, content, time_str, author

            except Exception as e:
                print(f"❌ JSON解析失败: {e}")

        return "", "", "", ""

    def find_article_data_in_json(self, data, depth=0):
        """递归查找JSON中的文章数据"""
        if depth > 10:  # 防止无限递归
            return None

        if isinstance(data, dict):
            # 查找文章相关的key
            article_keys = ['articleDetail', 'detail', 'article', 'news', 'content']
            for key in article_keys:
                if key in data:
                    article_data = data[key]
                    if isinstance(article_data, dict) and article_data.get('title'):
                        return article_data

            # 递归搜索
            for value in data.values():
                if isinstance(value, (dict, list)):
                    result = self.find_article_data_in_json(value, depth + 1)
                    if result:
                        return result

        elif isinstance(data, list):
            for item in data:
                if isinstance(item, (dict, list)):
                    result = self.find_article_data_in_json(item, depth + 1)
                    if result:
                        return result

        return None

    def extract_from_dom_elements(self, soup):
        """从DOM元素中提取"""
        title = ""
        content = ""
        time_str = ""
        author = ""

        # 提取标题
        title_selectors = [
            'h1',
            '.detail-title',
            '.article-title',
            '[class*="title"]'
        ]

        for selector in title_selectors:
            elements = soup.select(selector)
            for elem in elements:
                text = elem.get_text().strip()
                if 8 < len(text) < 200 and not any(skip in text for skip in ['财联社-主流', '关于我们']):
                    title = text
                    break
            if title:
                break

        # 提取内容
        content_selectors = [
            '.detail-content',
            '.article-content',
            '.content-main',
            '[class*="content"]'
        ]

        for selector in content_selectors:
            elements = soup.select(selector)
            for elem in elements:
                # 移除脚本和样式
                for script in elem.find_all(['script', 'style']):
                    script.decompose()
                text = elem.get_text().strip()
                if len(text) > 100:
                    content = text
                    break
            if content:
                break

        # 提取时间
        time_elements = soup.find_all(class_=re.compile(r'time|date'))
        for elem in time_elements:
            text = elem.get_text().strip()
            if re.search(r'\d{4}[-年]\d{1,2}[-月]\d{1,2}', text):
                time_str = text
                break

        # 提取作者
        author_elements = soup.find_all(class_=re.compile(r'author|reporter|editor'))
        for elem in author_elements:
            text = elem.get_text().strip()
            if text and len(text) < 20:
                author = text
                break

        print(f"📋 DOM提取: 标题={len(title)}字, 内容={len(content)}字")
        return title, content, time_str, author

    def extract_from_smart_text(self, html_content):
        """智能文本提取"""
        # 移除HTML标签
        text = re.sub(r'<[^>]+>', '\n', html_content)
        lines = [line.strip() for line in text.split('\n') if line.strip()]

        title = ""
        content = ""
        time_str = ""
        author = ""

        # 查找标题（通常在前面且长度适中）
        for i, line in enumerate(lines[:50]):
            if (8 <= len(line) <= 150 and
                    not line.startswith(('关于我们', '网站声明', '联系方式', 'http')) and
                    not line.endswith(('.com', '.cn')) and
                    '©' not in line and
                    any(char in line for char in ['、', '：', '，', '！', '？', '"', '"'])):
                title = line
                break

        # 查找内容（寻找较长的段落）
        content_lines = []
        in_content = False

        for line in lines:
            # 跳过导航等
            if any(skip in line for skip in ['关于我们', '网站声明', '联系方式', '帮助', '首页']):
                continue

            # 开始收集内容的标志
            if ('财联社' in line and ('讯' in line or '电' in line)) or \
                    (len(line) > 20 and ('。' in line or '，' in line)):
                in_content = True

            if in_content and len(line) > 15:
                content_lines.append(line)

            if len(content_lines) > 20:
                break

        content = '\n'.join(content_lines[:15])

        # 查找时间
        for line in lines[:100]:
            time_match = re.search(r'(\d{4}[-年]\d{1,2}[-月]\d{1,2}[日\s]*\d{1,2}:\d{2})', line)
            if time_match:
                time_str = time_match.group(1)
                break

        # 查找作者
        for line in lines[:100]:
            author_match = re.search(r'财联社[记者编辑]*\s*([^\s\n]+)', line)
            if author_match:
                author = author_match.group(1)
                break

        print(f"📋 智能提取: 标题={len(title)}字, 内容={len(content)}字")
        return title, content, time_str, author

    def clean_html_content(self, content):
        """清理HTML内容"""
        if not content:
            return ""

        # 移除HTML标签
        content = re.sub(r'<[^>]+>', '', content)
        # 解码HTML实体
        content = html.unescape(content)
        # 清理空白字符
        content = re.sub(r'\s+', ' ', content).strip()

        return content

    def convert_timestamp(self, timestamp):
        """转换时间戳"""
        if not timestamp or not isinstance(timestamp, (int, float)):
            return datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

        try:
            if timestamp > 1e10:  # 毫秒级时间戳
                timestamp = timestamp / 1000
            return datetime.datetime.fromtimestamp(timestamp).strftime('%Y-%m-%d %H:%M:%S')
        except (ValueError, OSError):
            return datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

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
                        page_timeout=30000,  # 减少超时时间
                        delay_before_return_html=5.0,
                        magic=False,  # 禁用magic模式
                        remove_overlay_elements=False,
                        process_iframes=False,
                        exclude_external_links=True,
                        js_only=False,
                        screenshot=False,
                        # 移除wait_until参数
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
        """爬取单篇新闻"""
        print(f"📰 正在爬取新闻: {url}")

        result = await self.simple_crawl_with_retry(url, crawler)

        if not result or not result.success:
            print(f"❌ 爬取失败: {url}")
            return {}

        # 检查是否被重定向到首页
        if self.is_redirected_to_homepage(result.html):
            print(f"⚠️  页面被重定向，尝试备用方法: {url}")
            result = await self.try_alternative_access(url, crawler)
            if not result:
                print(f"❌ 所有方法都失败: {url}")
                return {}

        # 提取文章内容
        article_data = self.extract_article_content_advanced(result.html, url)

        if not article_data['success']:
            print(f"⚠️  内容提取失败: {url}")
            return {}

        # 构建返回数据
        news_data = {
            'title': article_data['title'],
            'abstract': article_data['content'][:200] + '...' if len(article_data['content']) > 200 else article_data['content'],
            'date': article_data['time'],
            'link': url,
            'content': article_data['content'],
            'id': md5(url),
            'type': category,
            'author': article_data['author'],
            'read_number': 0,
            'time': datetime.datetime.now().strftime('%Y-%m-%d')
        }

        print(f"✅ 提取成功: {article_data['title'][:50]}...")
        return news_data

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
        crawler_config = self.create_enhanced_crawler_config()

        async with AsyncWebCrawler(**crawler_config) as crawler:
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