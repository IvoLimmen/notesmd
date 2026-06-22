window.addEventListener('DOMContentLoaded',function () {
    document.addEventListener('keydown', function (event) {

        if(event.key == 'e' && window.location.toString().indexOf("/view/") != -1) {
            if (document.activeElement != null && document.activeElement.localName === 'input') {
                return;
            }
            var location = window.location.toString().replace("/view/", "/edit/")
            window.location = location;
        }

        if(event.ctrlKey && event.key == 's' && window.location.toString().indexOf("/edit/") != -1) {
            event.preventDefault();
            document.forms[1].submit();
        }

        if (document.activeElement != null && document.activeElement.tagName != 'INPUT' && document.activeElement.tagName != 'TEXTAREA') {
            if(event.key == '/') {
                event.stopPropagation();
                event.preventDefault();            
                document.getElementsByName('search')[0].focus();
            }

            if(event.key == 'h') {
                event.stopPropagation();
                event.preventDefault();            
                var location = window.location.toString();
                var index = location.lastIndexOf('/');            
                var newloc = location.substring(0, index) + 'Index';
                window.location = newloc;
            }
        }
    });

    document.querySelectorAll('pre code').forEach((block) => {
        block.addEventListener("click", () => {
            navigator.clipboard.writeText(block.textContent);
        });
    });
});
