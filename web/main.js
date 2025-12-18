document.addEventListener('keydown', function (event) {
    if(event.key == 'e' && window.location.toString().indexOf("/view/") != -1) {
        var location = window.location.toString().replace("/view/", "/edit/")
        window.location = location;
    }
});