import os
import re

html_files = [
    'index.html',
    'first_page.html',
    'explore_page.html',
    'freq_page.html',
    'introduce_page.html',
    'personal_page.html',
    'social_page.html'
]

os.makedirs('css', exist_ok=True)
os.makedirs('js', exist_ok=True)

for file in html_files:
    if not os.path.exists(file):
        continue
    with open(file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Process CSS
    style_pattern = re.compile(r'<style>(.*?)</style>', re.DOTALL)
    style_matches = list(style_pattern.finditer(content))
    
    if style_matches:
        match = style_matches[0]
        css_content = match.group(1).strip()
        css_filename = file.replace('.html', '.css')
        with open(f'css/{css_filename}', 'w', encoding='utf-8') as f:
            f.write(css_content)
        
        # Replace the first style block with link
        content = content[:match.start()] + f'<link rel="stylesheet" href="css/{css_filename}">' + content[match.end():]
        
    # Process JS
    script_pattern = re.compile(r'<script>(.*?)</script>', re.DOTALL)
    script_matches = list(script_pattern.finditer(content))
    js_content_parts = []
    
    js_filename = file.replace('.html', '.js')
    has_extracted = False
    
    # Iterate in reverse to allow safe string slicing replacements
    for match in reversed(script_matches):
        inner_content = match.group(1)
        if 'document.write' not in inner_content:
            # We want to extract this
            js_content_parts.insert(0, inner_content.strip())
            
            # Replace the script tag with nothing (we will append the src script at the end)
            content = content[:match.start()] + '' + content[match.end():]
            has_extracted = True
            
    if has_extracted:
        with open(f'js/{js_filename}', 'w', encoding='utf-8') as f:
            f.write('\n\n'.join(js_content_parts))
        
        # Insert <script src="js/js_filename"></script> right before </body>
        if '</body>' in content:
            content = content.replace('</body>', f'<script src="js/{js_filename}"></script>\n</body>')
        else:
            content += f'\n<script src="js/{js_filename}"></script>'
            
    with open(file, 'w', encoding='utf-8') as f:
        f.write(content)

print("Extraction complete!")
