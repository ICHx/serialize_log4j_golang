import org.apache.log4j.Logger;
import java.util.stream.IntStream;

public class sender {

   /* Get actual class name to be printed on */
   static Logger log = Logger.getLogger(sender.class.getName());

   public static void main(String[] args) {
      IntStream.range(0,5).forEach(i -> log.info("Hello " + i));
      log.debug("Bye");
   }
}
