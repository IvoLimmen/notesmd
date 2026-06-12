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
            document.forms[0].submit()
        }
    });

    document.querySelectorAll('pre code').forEach((block) => {
        block.addEventListener("click", () => {
            navigator.clipboard.writeText(block.textContent);
        });
    });
});
