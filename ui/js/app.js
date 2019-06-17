'use strict';

// the control plane URL, replaced by actual value, that is, the value of 
// of $EKSPHEMERAL_URL on container image build:
var cpURL = 'EKSPHEMERAL_URL';

$(document).ready(function($){
  // list clusters when user clicks the refresh button:
  $('#clusters > h2').click(function (event) {
    clusters();
  });

  // show cluster details when user clicks 'Details'
  // note: since it's an dynamically added element, needs the .on() form:
  $('body').on('click', 'span.showdetails', function () {
    event.stopPropagation();
    event.stopImmediatePropagation();
    var cID = $(this).parent().attr('id');
    clusterdetail(cID);
  });
});

function clusters(){
  var ep = '/status/*';
  $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
  $.ajax({
    type: "GET",
    url: cpURL + ep,
    dataType: 'json',
    async: false,
    error: function (d) {
      console.info(d);
      $('#status').html('<div>control plane seems down</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        var consoleURL = "https://console.aws.amazon.com/eks/home";
        for (let i = 0; i < d.length; i++) {
          var cID = d[i];
          buffer += '<div id="' + cID + '">';
          buffer += ' <a href="' + consoleURL + '" target="_blank" rel="noopener">' + cID + '</a>';
          buffer += ' <span class="showdetails">Details…</span>';
          buffer += '<div class="cdetails"></div>';
          buffer += '</div>';
        }
        $('#clusterdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}

function clusterdetail(cID) {
  var ep = '/status/'+cID;
  $('#status').html('<img src="./img/standby.gif" alt="please wait" width="64px">');
  $.ajax({
    type: "GET",
    url: cpURL + ep,
    dataType: 'json',
    async: false,
    error: function (d) {
      console.info(d);
      $('#status').html('<div>control plane seems down</div>');
    },
    success: function (d) {
      if (d != null) {
        console.info(d);
        var buffer = '';
        buffer += '<div class="cdfield"><span class="cdtitle">Name:</span> ' + d.name + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Kubernetes version:</span> ' + d.kubeversion + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Number of worker nodes:</span> ' + d.numworkers + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Created at:</span> ' + convertTimestamp(d.created) + '</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">TTL:</span> ' + d.ttl + ' min left</div>';
        buffer += '<div class="cdfield"><span class="cdtitle">Owner:</span> <a href="mailto:' + d.owner + '">' + d.owner + '</a> notified on creation and 5 min before destruction</div>';
        $('#' + cID + ' .cdetails').html(buffer);
        $('#status').html('');
      }
    }
  })
}


// as per https://gist.github.com/kmaida/6045266

function convertTimestamp(timestamp) {
  // converts the passed timestamp to milliseconds 
  var d = new Date(timestamp * 1000),
    yyyy = d.getFullYear(),
    // Months are zero based, hence adding leading 0:
    mm = ('0' + (d.getMonth() + 1)).slice(-2),
    // Add leading 0:
    dd = ('0' + d.getDate()).slice(-2),
    hh = d.getHours(),
    h = hh,
    // Add leading 0:
    min = ('0' + d.getMinutes()).slice(-2),
    ampm = 'AM',
    time;
  if (hh > 12) {
    h = hh - 12;
    ampm = 'PM';
  } else if (hh === 12) {
    h = 12;
    ampm = 'PM';
  } else if (hh == 0) {
    h = 12;
  }
  time = yyyy + '-' + mm + '-' + dd + 'T' + h + ':' + min;
  return time;
}